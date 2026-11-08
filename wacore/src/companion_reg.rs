//! `companion_platform_id` + `companion_platform_display` emission.
//! Encoding only. Ported from oxidezap #592/#593/#595.
//!
//! The wire ids here are NOT the proto `DeviceProps.PlatformType` numbering:
//! they follow `WAWebCompanionRegClientUtils.DEVICE_PLATFORM`, a separate
//! table the server validates (Edge=2, Firefox=3, IE=4, Opera=5, Safari=6).

use waproto::whatsapp as wa;

/// Prefix `WAWebLinkDeviceQrcode` uses when iOS native-camera linking is on.
/// Concatenate with `make_qr_data` output to get a scannable deep-link URL.
pub const NATIVE_CAMERA_DEEP_LINK_PREFIX: &str = "https://wa.me/settings/linked_devices#";

/// Web codes follow `WAWebCompanionRegClientUtils.DEVICE_PLATFORM`.
/// Android letters need server-side attestation, so they're reachable
/// only through explicit opt-in.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, Hash)]
pub enum CompanionWebClientType {
    Chrome,
    Edge,
    Firefox,
    Ie,
    Opera,
    Safari,
    Electron,
    Uwp,
    /// Default fallback. The proto's `UNKNOWN` (wire `'0'`) is absent
    /// because WA Web never emits it from a real browser and the server
    /// rejects it.
    #[default]
    OtherWebClient,
    AndroidTablet,
    AndroidPhone,
    AndroidAmbiguous,
}

impl CompanionWebClientType {
    /// Single-byte ASCII id placed in `<companion_platform_id>` (and the
    /// QR string's 5th field).
    pub const fn wire_byte(self) -> u8 {
        match self {
            Self::Chrome => b'1',
            Self::Edge => b'2',
            Self::Firefox => b'3',
            Self::Ie => b'4',
            Self::Opera => b'5',
            Self::Safari => b'6',
            Self::Electron => b'7',
            Self::Uwp => b'8',
            Self::OtherWebClient => b'9',
            Self::AndroidTablet => b'd',
            Self::AndroidPhone => b'e',
            Self::AndroidAmbiguous => b'f',
        }
    }
}

impl std::fmt::Display for CompanionWebClientType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.wire_byte() as char)
    }
}

/// Browser label for `companion_platform_display`. Non-browser variants
/// fall back to "Chrome" because WA Web's `info().name` reports the
/// underlying Chromium renderer in those contexts. Mobile variants are
/// short-circuited by [`companion_platform_display`] before reaching here.
pub const fn companion_browser_name(ct: CompanionWebClientType) -> &'static str {
    use CompanionWebClientType as C;
    match ct {
        C::Chrome => "Chrome",
        C::Edge => "Edge",
        C::Firefox => "Firefox",
        C::Ie => "IE",
        C::Opera => "Opera",
        C::Safari => "Safari",
        C::Electron
        | C::Uwp
        | C::OtherWebClient
        | C::AndroidTablet
        | C::AndroidPhone
        | C::AndroidAmbiguous => "Chrome",
    }
}

/// Android maps to `Chrome` because that's what real WA Web on
/// Chrome-Android emits and what the server accepts; the Android
/// letters need attestation we can't fake from this crate, so they
/// stay behind `PairCodeOptions::platform_id`. iOS/AR/VR and the
/// proto's `UNKNOWN` collapse to `OtherWebClient` — `'0'` would be
/// server-rejected.
pub const fn companion_web_client_type_for_platform(
    pt: wa::device_props::PlatformType,
) -> CompanionWebClientType {
    use CompanionWebClientType as C;
    use wa::device_props::PlatformType as P;
    match pt {
        P::Chrome => C::Chrome,
        P::Firefox => C::Firefox,
        P::Ie => C::Ie,
        P::Opera => C::Opera,
        P::Safari => C::Safari,
        P::Edge => C::Edge,
        P::Desktop => C::Electron,
        P::Uwp => C::Uwp,
        P::AndroidPhone | P::AndroidTablet | P::AndroidAmbiguous => C::Chrome,
        // Wildcard keeps future proto variants server-safe instead of
        // breaking exhaustiveness when waproto regenerates.
        _ => C::OtherWebClient,
    }
}

pub fn companion_web_client_type_for_props(props: &wa::DeviceProps) -> CompanionWebClientType {
    props
        .platform_type
        .and_then(|v| wa::device_props::PlatformType::try_from(v).ok())
        .map(companion_web_client_type_for_platform)
        .unwrap_or_default()
}

/// Canonical OS label for the pair-code `companion_platform_display`.
///
/// WA Web only ever emits a real OS name from its UA parser
/// (`WAWebBrowserInfo().os` → `ua-parser-js` `getOS().name`). Unlike QR
/// pairing — which never sends this field and therefore tolerates an
/// arbitrary branding string in `DeviceProps::os` — the pair-code
/// `companion_hello` server **rejects a non-OS display with `bad-request`**.
/// So the OS component must come from a closed set of real OS names, never
/// the free-form branding string.
///
/// The set is deliberately small and conservative: the server is lenient
/// toward real OS names today (it also accepts `Ubuntu`, `Fedora`, `Mac`, …),
/// but collapsing everything to a guaranteed-accepted canonical value is
/// robust against that leniency changing. Anything unrecognized coerces to
/// [`Self::Linux`] (see [`Self::from_hint`]).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CompanionOs {
    Windows,
    MacOs,
    Linux,
    Android,
    Ios,
}

impl CompanionOs {
    /// The exact wire string, matching `ua-parser-js` `getOS().name` (what WA
    /// Web sends) — note the `"Mac OS"` and `"iOS"` spellings.
    pub const fn wire_str(self) -> &'static str {
        match self {
            Self::Windows => "Windows",
            Self::MacOs => "Mac OS",
            Self::Linux => "Linux",
            Self::Android => "Android",
            Self::Ios => "iOS",
        }
    }

    /// Classify a free-form OS hint into a known canonical OS, or `None` when
    /// it is not recognizably an OS (empty, or a branding label such as
    /// `"Veloz"`). Case-insensitive. iPad folds to [`Self::Ios`]: older UA
    /// parsers report iPad as `"iOS"`, so `"iPadOS"` is not a confirmed
    /// server-accepted value. Chrome OS and Linux distros fold to
    /// [`Self::Linux`].
    pub fn classify(os: &str) -> Option<Self> {
        let os = os.trim().to_ascii_lowercase();
        if os.is_empty() {
            None
        } else if os.contains("windows") {
            Some(Self::Windows)
        } else if os.contains("mac") || os.contains("osx") || os.contains("darwin") {
            Some(Self::MacOs)
        } else if os.contains("ipad")
            || os.contains("iphone")
            // Whole-word "ios" only, so "KaiOS" etc. don't false-match the substring.
            || os.split(|c: char| !c.is_ascii_alphanumeric())
                .any(|tok| tok == "ios")
        {
            Some(Self::Ios)
        } else if os.contains("android") {
            Some(Self::Android)
        } else if os.contains("linux")
            || os.contains("ubuntu")
            || os.contains("debian")
            || os.contains("fedora")
            || os.contains("chrome os")
            || os.contains("chromeos")
            || os.contains("chromium")
            // Whole-word for the short ambiguous ones so branding like "March"/
            // "Search"/"across" doesn't false-match (bare "Arch"/"CrOS" still do;
            // "Arch Linux"/"archlinux" are caught by the "linux" substring above).
            || os.split(|c: char| !c.is_ascii_alphanumeric())
                .any(|tok| tok == "arch" || tok == "cros")
        {
            Some(Self::Linux)
        } else {
            None
        }
    }

    /// Coerce a free-form `DeviceProps::os` into a server-safe canonical OS,
    /// defaulting an unrecognized/branding value to [`Self::Linux`].
    pub fn from_hint(os: &str) -> Self {
        Self::classify(os).unwrap_or(Self::Linux)
    }
}

/// Formats `<Browser> (<os>)` (Android client types → `Android (<os>)`),
/// mirroring `WAWebAltDeviceLinkingIq`, with `os` used **verbatim** — no
/// canonicalization. This is the escape hatch for an advanced caller that
/// overrides the display OS (e.g. to keep a real distro name like `"Ubuntu"`
/// the server accepts); the server validates the OS, so a non-OS string here
/// is rejected with `bad-request`. Most callers want
/// [`companion_platform_display`].
pub fn companion_platform_display_raw(ct: CompanionWebClientType, os: &str) -> String {
    use CompanionWebClientType as C;
    match ct {
        C::AndroidPhone | C::AndroidTablet | C::AndroidAmbiguous => {
            format!("Android ({os})")
        }
        _ => format!("{} ({})", companion_browser_name(ct), os),
    }
}

/// `companion_platform_display` body: `<Browser> (<OS>)` (Android client
/// types emit `Android (<OS>)`), mirroring `WAWebAltDeviceLinkingIq`. The OS
/// is canonicalized through [`CompanionOs`] because the pair-code server
/// rejects a non-OS string here with `bad-request`; an unrecognized/branding
/// `os` (or an empty one) becomes `Linux`.
pub fn companion_platform_display(ct: CompanionWebClientType, os: &str) -> String {
    companion_platform_display_raw(ct, CompanionOs::from_hint(os).wire_str())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn wire_byte_matches_wa_web() {
        assert_eq!(CompanionWebClientType::Chrome.wire_byte(), b'1');
        assert_eq!(CompanionWebClientType::Edge.wire_byte(), b'2');
        assert_eq!(CompanionWebClientType::Firefox.wire_byte(), b'3');
        assert_eq!(CompanionWebClientType::Ie.wire_byte(), b'4');
        assert_eq!(CompanionWebClientType::Opera.wire_byte(), b'5');
        assert_eq!(CompanionWebClientType::Safari.wire_byte(), b'6');
        assert_eq!(CompanionWebClientType::Electron.wire_byte(), b'7');
        assert_eq!(CompanionWebClientType::Uwp.wire_byte(), b'8');
        assert_eq!(CompanionWebClientType::OtherWebClient.wire_byte(), b'9');
    }

    #[test]
    fn wire_byte_matches_apk_for_mobile() {
        assert_eq!(CompanionWebClientType::AndroidTablet.wire_byte(), b'd');
        assert_eq!(CompanionWebClientType::AndroidPhone.wire_byte(), b'e');
        assert_eq!(CompanionWebClientType::AndroidAmbiguous.wire_byte(), b'f');
    }

    #[test]
    fn display_renders_wire_byte_as_char() {
        assert_eq!(format!("{}", CompanionWebClientType::Chrome), "1");
        assert_eq!(format!("{}", CompanionWebClientType::OtherWebClient), "9");
        assert_eq!(format!("{}", CompanionWebClientType::AndroidPhone), "e");
    }

    #[test]
    fn default_is_other_web_client_nine() {
        assert_eq!(
            CompanionWebClientType::default(),
            CompanionWebClientType::OtherWebClient
        );
    }

    #[test]
    fn browser_and_desktop_platform_types_map_to_their_variants() {
        use CompanionWebClientType as C;
        use wa::device_props::PlatformType as P;
        for (pt, expected) in [
            (P::Chrome, C::Chrome),
            (P::Firefox, C::Firefox),
            (P::Edge, C::Edge),
            (P::Safari, C::Safari),
            (P::Opera, C::Opera),
            (P::Ie, C::Ie),
            (P::Desktop, C::Electron),
            (P::Uwp, C::Uwp),
        ] {
            assert_eq!(companion_web_client_type_for_platform(pt), expected);
        }
    }

    #[test]
    fn android_collapses_to_chrome_and_unknown_to_other() {
        use CompanionWebClientType as C;
        use wa::device_props::PlatformType as P;
        for pt in [P::AndroidPhone, P::AndroidTablet, P::AndroidAmbiguous] {
            assert_eq!(companion_web_client_type_for_platform(pt), C::Chrome);
        }
        for pt in [P::Unknown, P::Ipad, P::IosPhone, P::Vr, P::CloudApi] {
            assert_eq!(
                companion_web_client_type_for_platform(pt),
                C::OtherWebClient
            );
        }
    }

    #[test]
    fn for_props_reads_platform_type_and_falls_back() {
        let chrome = wa::DeviceProps {
            platform_type: Some(wa::device_props::PlatformType::Chrome as i32),
            ..Default::default()
        };
        assert_eq!(
            companion_web_client_type_for_props(&chrome),
            CompanionWebClientType::Chrome
        );
        // Absent or unrecognized platform_type falls back to OtherWebClient.
        let empty = wa::DeviceProps::default();
        assert_eq!(
            companion_web_client_type_for_props(&empty),
            CompanionWebClientType::OtherWebClient
        );
        let bogus = wa::DeviceProps {
            platform_type: Some(999),
            ..Default::default()
        };
        assert_eq!(
            companion_web_client_type_for_props(&bogus),
            CompanionWebClientType::OtherWebClient
        );
    }

    #[test]
    fn os_wire_spellings_match_ua_parser() {
        assert_eq!(CompanionOs::Windows.wire_str(), "Windows");
        assert_eq!(CompanionOs::MacOs.wire_str(), "Mac OS");
        assert_eq!(CompanionOs::Linux.wire_str(), "Linux");
        assert_eq!(CompanionOs::Android.wire_str(), "Android");
        assert_eq!(CompanionOs::Ios.wire_str(), "iOS");
    }

    #[test]
    fn os_classification_covers_common_names() {
        assert_eq!(
            CompanionOs::classify("Windows 11"),
            Some(CompanionOs::Windows)
        );
        assert_eq!(CompanionOs::classify("macOS"), Some(CompanionOs::MacOs));
        assert_eq!(CompanionOs::classify("Mac OS X"), Some(CompanionOs::MacOs));
        assert_eq!(CompanionOs::classify("Linux"), Some(CompanionOs::Linux));
        assert_eq!(
            CompanionOs::classify("Ubuntu 24.04"),
            Some(CompanionOs::Linux)
        );
        assert_eq!(
            CompanionOs::classify("Android 15"),
            Some(CompanionOs::Android)
        );
        assert_eq!(CompanionOs::classify("iOS 18"), Some(CompanionOs::Ios));
        assert_eq!(CompanionOs::classify("iPadOS 26"), Some(CompanionOs::Ios));
        // Branding and empty hints are not OS names.
        assert_eq!(CompanionOs::classify("rust"), None);
        assert_eq!(CompanionOs::classify("Veloz"), None);
        assert_eq!(CompanionOs::classify("  "), None);
        // Unknown branding coerces to Linux.
        assert_eq!(CompanionOs::from_hint("rust"), CompanionOs::Linux);
    }

    #[test]
    fn display_canonicalizes_os_and_android_renders_android() {
        assert_eq!(
            companion_platform_display(CompanionWebClientType::Chrome, "Ubuntu"),
            "Chrome (Linux)"
        );
        assert_eq!(
            companion_platform_display(CompanionWebClientType::Safari, "macOS 15"),
            "Safari (Mac OS)"
        );
        assert_eq!(
            companion_platform_display(CompanionWebClientType::Chrome, "rust"),
            "Chrome (Linux)"
        );
        assert_eq!(
            companion_platform_display(CompanionWebClientType::AndroidPhone, "Android 15"),
            "Android (Android)"
        );
    }
}
