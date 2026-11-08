use crate::is_option_type;
use proc_macro::TokenStream;
use quote::quote;
use syn::{Data, DeriveInput, Fields, parse_macro_input};

pub fn derive_wire_enum(input: TokenStream) -> TokenStream {
    let input = parse_macro_input!(input as DeriveInput);

    let variants = match &input.data {
        Data::Enum(e) => e.variants.clone(),
        _ => {
            return syn::Error::new_spanned(&input.ident, "WireEnum can only be derived for enums")
                .to_compile_error()
                .into();
        }
    };

    let cfg = match parse_enum_level_wire(&input.attrs) {
        Ok(c) => c,
        Err(e) => return e.to_compile_error().into(),
    };

    match cfg.kind {
        WireKind::IntTagged => expand_wire_enum_int(&input.ident, &variants).into(),
        WireKind::StringTagged {
            discriminator,
            content,
        } => expand_wire_enum_tagged(&input, &variants, &discriminator, content.as_deref()).into(),
        WireKind::UnitString => expand_wire_enum_unit(&input.ident, &variants).into(),
    }
}

// ----- enum-level config -----

enum WireKind {
    UnitString,
    StringTagged {
        discriminator: String,
        content: Option<String>,
    },
    IntTagged,
}

struct WireEnumCfg {
    kind: WireKind,
}

fn parse_enum_level_wire(attrs: &[syn::Attribute]) -> syn::Result<WireEnumCfg> {
    let mut tag_field: Option<String> = None;
    let mut content_field: Option<String> = None;
    let mut kind_is_int = false;

    for attr in attrs {
        if !attr.path().is_ident("wire") {
            continue;
        }
        attr.parse_nested_meta(|meta| {
            if meta.path.is_ident("tag") {
                let lit: syn::LitStr = meta.value()?.parse()?;
                tag_field = Some(lit.value());
            } else if meta.path.is_ident("content") {
                let lit: syn::LitStr = meta.value()?.parse()?;
                content_field = Some(lit.value());
            } else if meta.path.is_ident("kind") {
                let lit: syn::LitStr = meta.value()?.parse()?;
                match lit.value().as_str() {
                    "int" => kind_is_int = true,
                    "string" => kind_is_int = false,
                    other => {
                        return Err(meta.error(format!(
                            "unknown wire kind {other:?}; expected \"string\" or \"int\""
                        )));
                    }
                }
            } else {
                return Err(meta.error("unknown attribute inside #[wire(...)]"));
            }
            Ok(())
        })?;
    }

    let kind = if kind_is_int {
        if tag_field.is_some() || content_field.is_some() {
            return Err(syn::Error::new_spanned(
                &attrs[0],
                "#[wire(kind = \"int\")] is incompatible with tagged string fields",
            ));
        }
        WireKind::IntTagged
    } else if let Some(discriminator) = tag_field {
        WireKind::StringTagged {
            discriminator,
            content: content_field,
        }
    } else if content_field.is_some() {
        return Err(syn::Error::new_spanned(
            &attrs[0],
            "#[wire(content = \"...\")] requires #[wire(tag = \"...\")]",
        ));
    } else {
        WireKind::UnitString
    };

    Ok(WireEnumCfg { kind })
}

// ----- variant-level helpers -----

enum VariantWire {
    Str(String),
    Int(i32),
}

struct VariantInfo {
    ident: syn::Ident,
    fields: Fields,
    wire: Option<VariantWire>,
    aliases: Vec<String>,
    is_default: bool,
    is_fallback: bool,
}

fn read_variant(v: &syn::Variant) -> syn::Result<VariantInfo> {
    let mut wire: Option<VariantWire> = None;
    let mut aliases: Vec<String> = Vec::new();
    let mut is_default = false;
    let mut is_fallback = false;

    for attr in &v.attrs {
        if attr.path().is_ident("wire_default") {
            is_default = true;
        } else if attr.path().is_ident("wire_fallback") {
            is_fallback = true;
        } else if attr.path().is_ident("wire_alias") {
            if let syn::Meta::NameValue(nv) = &attr.meta
                && let syn::Expr::Lit(syn::ExprLit {
                    lit: syn::Lit::Str(s),
                    ..
                }) = &nv.value
            {
                aliases.push(s.value());
            } else {
                return Err(syn::Error::new_spanned(
                    attr,
                    "expected #[wire_alias = \"...\"] with a string literal",
                ));
            }
        } else if attr.path().is_ident("wire") {
            // Variant-level #[wire = "..."] or #[wire = 101]
            if let syn::Meta::NameValue(nv) = &attr.meta {
                match &nv.value {
                    syn::Expr::Lit(syn::ExprLit {
                        lit: syn::Lit::Str(s),
                        ..
                    }) => wire = Some(VariantWire::Str(s.value())),
                    syn::Expr::Lit(syn::ExprLit {
                        lit: syn::Lit::Int(n),
                        ..
                    }) => {
                        // Reject out-of-range literals at macro parse time rather
                        // than silently wrapping with `as i32`.
                        let parsed: i32 = n.base10_parse().map_err(|_| {
                            syn::Error::new_spanned(
                                n,
                                format!(
                                    "#[wire = {}] does not fit in i32 ({}..={})",
                                    n,
                                    i32::MIN,
                                    i32::MAX
                                ),
                            )
                        })?;
                        wire = Some(VariantWire::Int(parsed));
                    }
                    _ => {
                        return Err(syn::Error::new_spanned(
                            &nv.value,
                            "#[wire = ...] expects a string or integer literal",
                        ));
                    }
                }
            }
        }
    }

    Ok(VariantInfo {
        ident: v.ident.clone(),
        fields: v.fields.clone(),
        wire,
        aliases,
        is_default,
        is_fallback,
    })
}

fn field_has_wire_skip(attrs: &[syn::Attribute]) -> bool {
    for attr in attrs {
        if !attr.path().is_ident("wire") {
            continue;
        }
        let mut found_skip = false;
        let _ = attr.parse_nested_meta(|meta| {
            if meta.path.is_ident("skip") {
                found_skip = true;
            }
            Ok(())
        });
        if found_skip {
            return true;
        }
    }
    false
}

// ================== unit-string mode ==================

fn expand_wire_enum_unit(
    name: &syn::Ident,
    variants: &syn::punctuated::Punctuated<syn::Variant, syn::Token![,]>,
) -> proc_macro2::TokenStream {
    let mut infos = Vec::with_capacity(variants.len());
    for v in variants {
        match read_variant(v) {
            Ok(info) => infos.push(info),
            Err(e) => return e.to_compile_error(),
        }
    }

    let mut seen: std::collections::HashMap<String, syn::Ident> = Default::default();
    let mut fallback: Option<&VariantInfo> = None;
    let mut default_variant: Option<&VariantInfo> = None;

    for info in &infos {
        if info.is_fallback {
            if fallback.is_some() {
                return syn::Error::new_spanned(
                    &info.ident,
                    "only one #[wire_fallback] variant is allowed",
                )
                .to_compile_error();
            }
            match &info.fields {
                Fields::Unnamed(fields) if fields.unnamed.len() == 1 => {}
                _ => {
                    return syn::Error::new_spanned(
                        &info.ident,
                        "#[wire_fallback] on a unit-string enum requires VariantName(String)",
                    )
                    .to_compile_error();
                }
            }
            if info.wire.is_some() {
                return syn::Error::new_spanned(
                    &info.ident,
                    "#[wire_fallback] variant must not carry #[wire = \"...\"]",
                )
                .to_compile_error();
            }
            fallback = Some(info);
            if info.is_default {
                default_variant = Some(info);
            }
            continue;
        }
        if !matches!(info.fields, Fields::Unit) {
            return syn::Error::new_spanned(
                &info.ident,
                "unit-string WireEnum only supports unit variants (use #[wire_fallback] for a catch-all)",
            )
            .to_compile_error();
        }
        let Some(VariantWire::Str(s)) = &info.wire else {
            return syn::Error::new_spanned(&info.ident, "variant needs #[wire = \"...\"]")
                .to_compile_error();
        };
        if let Some(prev) = seen.insert(s.clone(), info.ident.clone()) {
            return syn::Error::new_spanned(
                &info.ident,
                format!("duplicate #[wire = \"{s}\"]; already used by {prev}"),
            )
            .to_compile_error();
        }
        if info.is_default {
            if default_variant.is_some() {
                return syn::Error::new_spanned(&info.ident, "only one #[wire_default] is allowed")
                    .to_compile_error();
            }
            default_variant = Some(info);
        }
        for alias in &info.aliases {
            if let Some(prev) = seen.insert(alias.clone(), info.ident.clone()) {
                return syn::Error::new_spanned(
                    &info.ident,
                    format!(
                        "#[wire_alias = \"{alias}\"] collides with existing wire tag from variant {prev}"
                    ),
                )
                .to_compile_error();
            }
        }
    }

    let first_known: Option<&VariantInfo> = infos.iter().find(|i| !i.is_fallback);
    let default_info = match (default_variant, first_known, fallback) {
        (Some(d), _, _) => d,
        (None, Some(f), _) => f,
        (None, None, Some(fb)) => fb,
        (None, None, None) => {
            return syn::Error::new_spanned(name, "WireEnum cannot be derived for empty enums")
                .to_compile_error();
        }
    };
    let default_ident = &default_info.ident;
    let default_ctor = if default_info.is_fallback {
        quote! { #name::#default_ident(::std::string::String::new()) }
    } else {
        quote! { #name::#default_ident }
    };

    let known: Vec<(&syn::Ident, &String)> = infos
        .iter()
        .filter(|i| !i.is_fallback)
        .map(|i| {
            let VariantWire::Str(s) = i.wire.as_ref().unwrap() else {
                unreachable!()
            };
            (&i.ident, s)
        })
        .collect();

    // `as_str()` always returns the PRIMARY tag — aliases are parser-only and
    // must never surface in serialization.
    let as_str_arms: Vec<_> = known
        .iter()
        .map(|(id, s)| quote! { #name::#id => #s })
        .collect();

    // For parsing, include primary + each alias; all map to the same variant.
    let try_from_arms: Vec<proc_macro2::TokenStream> = infos
        .iter()
        .filter(|i| !i.is_fallback)
        .flat_map(|i| {
            let id = &i.ident;
            let VariantWire::Str(primary) = i.wire.as_ref().unwrap() else {
                unreachable!()
            };
            std::iter::once(primary.clone())
                .chain(i.aliases.iter().cloned())
                .map(move |s| quote! { #s => ::core::result::Result::Ok(#name::#id) })
        })
        .collect();

    let from_arms: Vec<proc_macro2::TokenStream> = infos
        .iter()
        .filter(|i| !i.is_fallback)
        .flat_map(|i| {
            let id = &i.ident;
            let VariantWire::Str(primary) = i.wire.as_ref().unwrap() else {
                unreachable!()
            };
            std::iter::once(primary.clone())
                .chain(i.aliases.iter().cloned())
                .map(move |s| quote! { #s => #name::#id })
        })
        .collect();

    let as_str_return_ty;
    let as_str_block;
    let conversion_impls;

    if let Some(fb) = fallback {
        let fb_ident = &fb.ident;
        as_str_return_ty = quote! { &str };
        as_str_block = quote! {
            match self {
                #(#as_str_arms,)*
                #name::#fb_ident(s) => s.as_str(),
            }
        };
        conversion_impls = quote! {
            impl ::core::convert::From<&str> for #name {
                fn from(value: &str) -> Self {
                    match value {
                        #(#from_arms,)*
                        other => #name::#fb_ident(other.to_string()),
                    }
                }
            }

        };
    } else {
        as_str_return_ty = quote! { &'static str };
        as_str_block = quote! {
            match self {
                #(#as_str_arms),*
            }
        };
        conversion_impls = quote! {
            impl ::core::convert::TryFrom<&str> for #name {
                type Error = ::anyhow::Error;
                fn try_from(value: &str) -> ::core::result::Result<Self, Self::Error> {
                    match value {
                        #(#try_from_arms),*,
                        _ => ::core::result::Result::Err(
                            ::anyhow::anyhow!("unknown {}: {}", stringify!(#name), value)
                        ),
                    }
                }
            }

        };
    }

    let deserialize_impl = if let Some(fb) = fallback {
        let fb_ident = &fb.ident;
        quote! {
            impl<'de> ::serde::Deserialize<'de> for #name {
                fn deserialize<D: ::serde::Deserializer<'de>>(
                    deserializer: D,
                ) -> ::core::result::Result<Self, D::Error> {
                    struct WireEnumVisitor;

                    impl<'de> ::serde::de::Visitor<'de> for WireEnumVisitor {
                        type Value = #name;

                        fn expecting(
                            &self,
                            formatter: &mut ::core::fmt::Formatter<'_>,
                        ) -> ::core::fmt::Result {
                            formatter.write_str("a wire enum string")
                        }

                        fn visit_str<E>(
                            self,
                            value: &str,
                        ) -> ::core::result::Result<Self::Value, E>
                        where
                            E: ::serde::de::Error,
                        {
                            ::core::result::Result::Ok(
                                <#name as ::core::convert::From<&str>>::from(value),
                            )
                        }

                        fn visit_string<E>(
                            self,
                            value: ::std::string::String,
                        ) -> ::core::result::Result<Self::Value, E>
                        where
                            E: ::serde::de::Error,
                        {
                            ::core::result::Result::Ok(match value.as_str() {
                                #(#from_arms,)*
                                _ => #name::#fb_ident(value),
                            })
                        }
                    }

                    deserializer.deserialize_str(WireEnumVisitor)
                }
            }
        }
    } else {
        quote! {
            impl<'de> ::serde::Deserialize<'de> for #name {
                fn deserialize<D: ::serde::Deserializer<'de>>(
                    deserializer: D,
                ) -> ::core::result::Result<Self, D::Error> {
                    struct WireEnumVisitor;

                    impl<'de> ::serde::de::Visitor<'de> for WireEnumVisitor {
                        type Value = #name;

                        fn expecting(
                            &self,
                            formatter: &mut ::core::fmt::Formatter<'_>,
                        ) -> ::core::fmt::Result {
                            formatter.write_str("a known wire enum string")
                        }

                        fn visit_str<E>(
                            self,
                            value: &str,
                        ) -> ::core::result::Result<Self::Value, E>
                        where
                            E: ::serde::de::Error,
                        {
                            <#name as ::core::convert::TryFrom<&str>>::try_from(value)
                                .map_err(E::custom)
                        }
                    }

                    deserializer.deserialize_str(WireEnumVisitor)
                }
            }
        }
    };

    quote! {
        impl #name {
            /// Wire string for this variant (single source of truth).
            pub fn as_str(&self) -> #as_str_return_ty {
                #as_str_block
            }
        }

        impl ::core::fmt::Display for #name {
            fn fmt(&self, f: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
                f.write_str(self.as_str())
            }
        }

        #conversion_impls

        impl ::core::default::Default for #name {
            fn default() -> Self {
                #default_ctor
            }
        }

        impl ::serde::Serialize for #name {
            fn serialize<S: ::serde::Serializer>(
                &self,
                serializer: S,
            ) -> ::core::result::Result<S::Ok, S::Error> {
                serializer.serialize_str(self.as_str())
            }
        }

        #deserialize_impl
    }
}

// ================== int mode ==================

fn expand_wire_enum_int(
    name: &syn::Ident,
    variants: &syn::punctuated::Punctuated<syn::Variant, syn::Token![,]>,
) -> proc_macro2::TokenStream {
    let mut infos = Vec::with_capacity(variants.len());
    for v in variants {
        match read_variant(v) {
            Ok(info) => infos.push(info),
            Err(e) => return e.to_compile_error(),
        }
    }

    let mut fallback: Option<&VariantInfo> = None;
    let mut seen: std::collections::HashMap<i32, syn::Ident> = Default::default();

    for info in &infos {
        if info.is_fallback {
            if fallback.is_some() {
                return syn::Error::new_spanned(
                    &info.ident,
                    "only one #[wire_fallback] is allowed",
                )
                .to_compile_error();
            }
            match &info.fields {
                Fields::Unnamed(fields) if fields.unnamed.len() == 1 => {}
                _ => {
                    return syn::Error::new_spanned(
                        &info.ident,
                        "#[wire_fallback] in int mode requires VariantName(i32)",
                    )
                    .to_compile_error();
                }
            }
            fallback = Some(info);
            continue;
        }
        if !matches!(info.fields, Fields::Unit) {
            return syn::Error::new_spanned(
                &info.ident,
                "int-mode WireEnum variants must be unit variants (except the #[wire_fallback])",
            )
            .to_compile_error();
        }
        let Some(VariantWire::Int(n)) = &info.wire else {
            return syn::Error::new_spanned(&info.ident, "variant needs #[wire = NUMBER]")
                .to_compile_error();
        };
        if let Some(prev) = seen.insert(*n, info.ident.clone()) {
            return syn::Error::new_spanned(
                &info.ident,
                format!("duplicate #[wire = {n}]; already used by {prev}"),
            )
            .to_compile_error();
        }
    }

    let code_arms: Vec<_> = infos
        .iter()
        .filter(|i| !i.is_fallback)
        .map(|i| {
            let id = &i.ident;
            let Some(VariantWire::Int(n)) = i.wire.as_ref() else {
                unreachable!()
            };
            let lit = proc_macro2::Literal::i32_suffixed(*n);
            quote! { #name::#id => #lit }
        })
        .collect();

    let from_arms: Vec<_> = infos
        .iter()
        .filter(|i| !i.is_fallback)
        .map(|i| {
            let id = &i.ident;
            let Some(VariantWire::Int(n)) = i.wire.as_ref() else {
                unreachable!()
            };
            let lit = proc_macro2::Literal::i32_suffixed(*n);
            quote! { #lit => #name::#id }
        })
        .collect();

    let strict_from_arms: Vec<_> = infos
        .iter()
        .filter(|i| !i.is_fallback)
        .map(|i| {
            let id = &i.ident;
            let Some(VariantWire::Int(n)) = i.wire.as_ref() else {
                unreachable!()
            };
            let lit = proc_macro2::Literal::i32_suffixed(*n);
            quote! { #lit => ::core::result::Result::Ok(#name::#id) }
        })
        .collect();

    let conversion = if let Some(fallback) = fallback {
        let fallback_ident = &fallback.ident;
        quote! {
            impl ::core::convert::From<i32> for #name {
                fn from(code: i32) -> Self {
                    match code {
                        #(#from_arms,)*
                        other => #name::#fallback_ident(other),
                    }
                }
            }
        }
    } else {
        quote! {
            impl ::core::convert::TryFrom<i32> for #name {
                type Error = i32;

                fn try_from(code: i32) -> ::core::result::Result<Self, Self::Error> {
                    match code {
                        #(#strict_from_arms,)*
                        other => ::core::result::Result::Err(other),
                    }
                }
            }
        }
    };

    let fallback_code_arm = fallback.map(|fallback| {
        let fallback_ident = &fallback.ident;
        quote! { #name::#fallback_ident(n) => *n, }
    });

    let deserialize = if fallback.is_some() {
        quote! {
            ::core::result::Result::Ok(<Self as ::core::convert::From<i32>>::from(n))
        }
    } else {
        quote! {
            <Self as ::core::convert::TryFrom<i32>>::try_from(n).map_err(|unknown| {
                <D::Error as ::serde::de::Error>::custom(::core::format_args!(
                    "unknown numeric wire code {unknown} for {}",
                    ::core::stringify!(#name),
                ))
            })
        }
    };

    quote! {
        impl #name {
            /// Numeric wire code for this variant (single source of truth).
            pub fn code(&self) -> i32 {
                match self {
                    #(#code_arms,)*
                    #fallback_code_arm
                }
            }
        }

        #conversion

        impl ::serde::Serialize for #name {
            fn serialize<S: ::serde::Serializer>(
                &self,
                serializer: S,
            ) -> ::core::result::Result<S::Ok, S::Error> {
                serializer.serialize_i32(self.code())
            }
        }

        impl<'de> ::serde::Deserialize<'de> for #name {
            fn deserialize<D: ::serde::Deserializer<'de>>(
                deserializer: D,
            ) -> ::core::result::Result<Self, D::Error> {
                let n = <i32 as ::serde::Deserialize>::deserialize(deserializer)?;
                #deserialize
            }
        }
    }
}

// ================== tagged mode ==================

fn expand_wire_enum_tagged(
    input: &DeriveInput,
    variants: &syn::punctuated::Punctuated<syn::Variant, syn::Token![,]>,
    discriminator: &str,
    content: Option<&str>,
) -> proc_macro2::TokenStream {
    let name = &input.ident;
    let visibility = &input.vis;
    let non_exhaustive = input
        .attrs
        .iter()
        .filter(|attr| attr.path().is_ident("non_exhaustive"));
    let mut infos = Vec::with_capacity(variants.len());
    for v in variants {
        match read_variant(v) {
            Ok(info) => infos.push(info),
            Err(e) => return e.to_compile_error(),
        }
    }

    let mut seen: std::collections::HashMap<String, syn::Ident> = Default::default();
    let mut fallback: Option<&VariantInfo> = None;

    for info in &infos {
        if info.is_fallback {
            if fallback.is_some() {
                return syn::Error::new_spanned(
                    &info.ident,
                    "only one #[wire_fallback] is allowed",
                )
                .to_compile_error();
            }
            // Must be { tag: String }
            let ok = matches!(
                &info.fields,
                Fields::Named(n)
                    if n.named.len() == 1
                        && n.named
                            .first()
                            .unwrap()
                            .ident
                            .as_ref()
                            .map(|i| i == "tag")
                            .unwrap_or(false)
            );
            if !ok {
                return syn::Error::new_spanned(
                    &info.ident,
                    "tagged #[wire_fallback] must have exactly { tag: String }",
                )
                .to_compile_error();
            }
            if info.wire.is_some() {
                return syn::Error::new_spanned(
                    &info.ident,
                    "#[wire_fallback] variant must not have #[wire = \"...\"]",
                )
                .to_compile_error();
            }
            fallback = Some(info);
            continue;
        }
        let Some(VariantWire::Str(s)) = &info.wire else {
            return syn::Error::new_spanned(&info.ident, "variant needs #[wire = \"...\"]")
                .to_compile_error();
        };
        if let Some(prev) = seen.insert(s.clone(), info.ident.clone()) {
            return syn::Error::new_spanned(
                &info.ident,
                format!("duplicate #[wire = \"{s}\"]; already used by {prev}"),
            )
            .to_compile_error();
        }
        for alias in &info.aliases {
            if let Some(prev) = seen.insert(alias.clone(), info.ident.clone()) {
                return syn::Error::new_spanned(
                    &info.ident,
                    format!(
                        "#[wire_alias = \"{alias}\"] collides with wire tag from variant {prev}"
                    ),
                )
                .to_compile_error();
            }
        }
    }

    if content.is_some() {
        if let Some(info) = fallback {
            return syn::Error::new_spanned(
                &info.ident,
                "adjacently tagged WireEnum does not support #[wire_fallback]",
            )
            .to_compile_error();
        }
        for info in &infos {
            let fields = match &info.fields {
                Fields::Named(fields) => &fields.named,
                Fields::Unnamed(fields) => &fields.unnamed,
                Fields::Unit => continue,
            };
            if let Some(field) = fields
                .iter()
                .find(|field| field_has_wire_skip(&field.attrs))
            {
                return syn::Error::new_spanned(
                    field,
                    "adjacently tagged WireEnum does not support #[wire(skip)]",
                )
                .to_compile_error();
            }
        }
    }

    // --- wire_tag(&self) -> &str ---

    let wire_tag_arms: Vec<_> = infos
        .iter()
        .map(|info| {
            let id = &info.ident;
            if info.is_fallback {
                // { tag: String } — return borrowed from the field
                quote! { #name::#id { tag } => tag.as_str() }
            } else {
                let VariantWire::Str(s) = info.wire.as_ref().unwrap() else {
                    unreachable!()
                };
                match &info.fields {
                    Fields::Unit => quote! { #name::#id => #s },
                    Fields::Named(_) => quote! { #name::#id { .. } => #s },
                    Fields::Unnamed(_) => quote! { #name::#id(..) => #s },
                }
            }
        })
        .collect();
    let wire_tag_return = if fallback.is_some() {
        quote! { &str }
    } else {
        quote! { &'static str }
    };

    // --- Serde implementation ---

    let discriminator_lit = discriminator;
    let serde_impl = if let Some(content_lit) = content {
        let borrowed_ident = quote::format_ident!("__{}WireBorrowed", name);
        let owned_ident = quote::format_ident!("__{}WireOwned", name);

        let borrowed_variants: Vec<_> = infos
            .iter()
            .map(|info| {
                let id = &info.ident;
                let VariantWire::Str(primary) = info.wire.as_ref().unwrap() else {
                    unreachable!()
                };
                let aliases = info
                    .aliases
                    .iter()
                    .map(|alias| quote! { #[serde(alias = #alias)] });
                let fields = match &info.fields {
                    Fields::Unit => quote! {},
                    Fields::Named(fields) => {
                        let declarations = fields.named.iter().map(|field| {
                            let field_id = field.ident.as_ref().unwrap();
                            let ty = &field.ty;
                            quote! { #field_id: &'__wire #ty }
                        });
                        quote! { { #(#declarations),* } }
                    }
                    Fields::Unnamed(fields) => {
                        let declarations = fields.unnamed.iter().map(|field| {
                            let ty = &field.ty;
                            quote! { &'__wire #ty }
                        });
                        quote! { ( #(#declarations),* ) }
                    }
                };
                quote! {
                    #[serde(rename = #primary)]
                    #(#aliases)*
                    #id #fields
                }
            })
            .collect();

        let owned_variants: Vec<_> = infos
            .iter()
            .map(|info| {
                let id = &info.ident;
                let VariantWire::Str(primary) = info.wire.as_ref().unwrap() else {
                    unreachable!()
                };
                let aliases = info
                    .aliases
                    .iter()
                    .map(|alias| quote! { #[serde(alias = #alias)] });
                let fields = match &info.fields {
                    Fields::Unit => quote! {},
                    Fields::Named(fields) => {
                        let declarations = fields.named.iter().map(|field| {
                            let field_id = field.ident.as_ref().unwrap();
                            let ty = &field.ty;
                            quote! { #field_id: #ty }
                        });
                        quote! { { #(#declarations),* } }
                    }
                    Fields::Unnamed(fields) => {
                        let declarations = fields.unnamed.iter().map(|field| {
                            let ty = &field.ty;
                            quote! { #ty }
                        });
                        quote! { ( #(#declarations),* ) }
                    }
                };
                quote! {
                    #[serde(rename = #primary)]
                    #(#aliases)*
                    #id #fields
                }
            })
            .collect();

        let borrow_arms: Vec<_> = infos
            .iter()
            .map(|info| {
                let id = &info.ident;
                match &info.fields {
                    Fields::Unit => {
                        quote! { #name::#id => #borrowed_ident::#id }
                    }
                    Fields::Named(fields) => {
                        let bindings = fields
                            .named
                            .iter()
                            .map(|field| field.ident.as_ref().unwrap());
                        let values = bindings.clone();
                        quote! {
                            #name::#id { #(#bindings),* } =>
                                #borrowed_ident::#id { #(#values),* }
                        }
                    }
                    Fields::Unnamed(fields) => {
                        let bindings: Vec<_> = (0..fields.unnamed.len())
                            .map(|index| quote::format_ident!("__field_{index}"))
                            .collect();
                        let values = &bindings;
                        quote! {
                            #name::#id(#(#bindings),*) =>
                                #borrowed_ident::#id(#(#values),*)
                        }
                    }
                }
            })
            .collect();

        let own_arms: Vec<_> = infos
            .iter()
            .map(|info| {
                let id = &info.ident;
                match &info.fields {
                    Fields::Unit => quote! { #owned_ident::#id => #name::#id },
                    Fields::Named(fields) => {
                        let bindings = fields
                            .named
                            .iter()
                            .map(|field| field.ident.as_ref().unwrap());
                        let values = bindings.clone();
                        quote! {
                            #owned_ident::#id { #(#bindings),* } =>
                                #name::#id { #(#values),* }
                        }
                    }
                    Fields::Unnamed(fields) => {
                        let bindings: Vec<_> = (0..fields.unnamed.len())
                            .map(|index| quote::format_ident!("__field_{index}"))
                            .collect();
                        let values = &bindings;
                        quote! {
                            #owned_ident::#id(#(#bindings),*) =>
                                #name::#id(#(#values),*)
                        }
                    }
                }
            })
            .collect();

        quote! {
            #[doc(hidden)]
            #[derive(::serde::Serialize)]
            #[serde(tag = #discriminator_lit, content = #content_lit)]
            enum #borrowed_ident<'__wire> {
                #(#borrowed_variants),*
            }

            #[doc(hidden)]
            #[derive(::serde::Deserialize)]
            #[serde(tag = #discriminator_lit, content = #content_lit)]
            enum #owned_ident {
                #(#owned_variants),*
            }

            impl ::serde::Serialize for #name {
                fn serialize<S: ::serde::Serializer>(
                    &self,
                    serializer: S,
                ) -> ::core::result::Result<S::Ok, S::Error> {
                    let value = match self {
                        #(#borrow_arms),*
                    };
                    ::serde::Serialize::serialize(&value, serializer)
                }
            }

            impl<'de> ::serde::Deserialize<'de> for #name {
                fn deserialize<D: ::serde::Deserializer<'de>>(
                    deserializer: D,
                ) -> ::core::result::Result<Self, D::Error> {
                    let value = <#owned_ident as ::serde::Deserialize>::deserialize(deserializer)?;
                    ::core::result::Result::Ok(match value {
                        #(#own_arms),*
                    })
                }
            }
        }
    } else {
        let struct_name_lit = name.to_string();
        // Struct-field path, not map keys: serializers that intern struct keys
        // cannot intern a name that arrives as a value.
        let serialize_arms: Vec<_> = infos
            .iter()
            .map(|info| {
                let id = &info.ident;
                let emit_arm = |pattern: proc_macro2::TokenStream,
                                len_prelude: proc_macro2::TokenStream,
                                entries: Vec<proc_macro2::TokenStream>| {
                    quote! {
                        #pattern => {
                            #len_prelude
                            let mut __state = ::serde::Serializer::serialize_struct(
                                serializer, #struct_name_lit, __len
                            )?;
                            ::serde::ser::SerializeStruct::serialize_field(
                                &mut __state, #discriminator_lit, self.wire_tag()
                            )?;
                            #(#entries)*
                            ::serde::ser::SerializeStruct::end(__state)
                        }
                    }
                };
                if info.is_fallback {
                    emit_arm(
                        quote! { #name::#id { tag: _ } },
                        quote! { let __len = 1usize; },
                        Vec::new(),
                    )
                } else {
                    match &info.fields {
                        Fields::Unit => emit_arm(
                            quote! { #name::#id },
                            quote! { let __len = 1usize; },
                            Vec::new(),
                        ),
                        Fields::Named(named) => {
                            let bindings: Vec<proc_macro2::TokenStream> = named
                                .named
                                .iter()
                                .map(|field| {
                                    let id = field.ident.as_ref().unwrap();
                                    if field_has_wire_skip(&field.attrs) {
                                        quote! { #id: _ }
                                    } else {
                                        quote! { #id }
                                    }
                                })
                                .collect();
                            let serialized: Vec<&syn::Field> = named
                                .named
                                .iter()
                                .filter(|field| !field_has_wire_skip(&field.attrs))
                                .collect();
                            let optional: Vec<&syn::Field> = serialized
                                .iter()
                                .copied()
                                .filter(|field| is_option_type(&field.ty))
                                .collect();
                            // Exact, not an upper bound: length-prefixed formats
                            // encode this count.
                            let constant_count = serialized.len() - optional.len() + 1;
                            let len_prelude = if optional.is_empty() {
                                quote! { let __len = #constant_count; }
                            } else {
                                let increments = optional.iter().map(|field| {
                                    let id = field.ident.as_ref().unwrap();
                                    quote! {
                                        if ::core::option::Option::is_some(#id) {
                                            __len += 1;
                                        }
                                    }
                                });
                                quote! {
                                    let mut __len = #constant_count;
                                    #(#increments)*
                                }
                            };
                            let entries: Vec<proc_macro2::TokenStream> = serialized
                                .iter()
                                .map(|field| {
                                    let id = field.ident.as_ref().unwrap();
                                    let key = id.to_string();
                                    if is_option_type(&field.ty) {
                                        quote! {
                                            if let ::core::option::Option::Some(__v) = #id {
                                                ::serde::ser::SerializeStruct::serialize_field(
                                                    &mut __state, #key, __v
                                                )?;
                                            }
                                        }
                                    } else {
                                        quote! {
                                            ::serde::ser::SerializeStruct::serialize_field(
                                                &mut __state, #key, #id
                                            )?;
                                        }
                                    }
                                })
                                .collect();
                            emit_arm(
                                quote! { #name::#id { #(#bindings),* } },
                                len_prelude,
                                entries,
                            )
                        }
                        // Scoped to the offending variant so it cannot shadow
                        // the arms that follow it.
                        Fields::Unnamed(_) => quote! {
                            #name::#id(..) => compile_error!("tagged WireEnum tuple variants require #[wire(content = \"...\")]")
                        },
                    }
                }
            })
            .collect();

        quote! {
            impl ::serde::Serialize for #name {
                fn serialize<S: ::serde::Serializer>(
                    &self,
                    serializer: S,
                ) -> ::core::result::Result<S::Ok, S::Error> {
                    match self {
                        #(#serialize_arms,)*
                    }
                }
            }
        }
    };

    // --- Sibling <Name>Tag unit enum (unit-string WireEnum) ---
    // The dispatch API must have the same visibility and room to grow as its source.

    let tag_ident = quote::format_ident!("{}Tag", name);

    let mut tag_variant_tokens: Vec<proc_macro2::TokenStream> = Vec::new();
    for info in &infos {
        let id = &info.ident;
        if info.is_fallback {
            tag_variant_tokens.push(quote! {
                #[doc = "Unknown wire tag — captured for forward compatibility."]
                #[wire_fallback]
                Unknown(::std::string::String)
            });
            continue;
        }
        let VariantWire::Str(primary) = info.wire.as_ref().unwrap() else {
            unreachable!()
        };
        // Primary tag + aliases collapse into ONE tag variant. The unit-string
        // WireEnum derive on the tag enum expands `#[wire_alias = "..."]` into
        // extra `From<&str>` arms pointing at the same variant, so parsers see
        // `Tag::Foo` regardless of whether the wire tag was the primary or an
        // alias.
        let alias_attrs = info.aliases.iter().map(|a| quote! { #[wire_alias = #a] });
        tag_variant_tokens.push(quote! {
            #[wire = #primary]
            #(#alias_attrs)*
            #id
        });
    }

    // --- Final expansion ---

    quote! {
        impl #name {
            /// The wire tag this variant serializes as — the JSON discriminator
            /// and the exact tag the parser dispatches on.
            pub fn wire_tag(&self) -> #wire_tag_return {
                match self {
                    #(#wire_tag_arms,)*
                }
            }

            /// Back-compat alias of [`Self::wire_tag`].
            #[inline]
            pub fn tag_name(&self) -> &str {
                self.wire_tag()
            }
        }

        #serde_impl

        /// Sibling unit enum listing every canonical wire tag for parser
        /// dispatch. Primary wire tags and any `#[wire_alias]` entries all
        /// resolve to the same variant via `From<&str>`.
        #[doc = "Auto-generated by `#[derive(WireEnum)]`."]
        #[derive(Debug, Clone, PartialEq, Eq, ::wacore::WireEnum)]
        #[allow(clippy::enum_variant_names)]
        #(#non_exhaustive)*
        #visibility enum #tag_ident {
            #(#tag_variant_tokens,)*
        }
    }
}
