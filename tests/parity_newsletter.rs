use wa_rs::features::newsletter::{NewsletterMetadata, NewsletterRole, NewsletterState};
#[test]
fn newsletter_types_are_stable() {
    let metadata = NewsletterMetadata {
        jid: "a@newsletter".into(),
        name: "n".into(),
        state: NewsletterState::Active,
    };
    assert_eq!(metadata.state, NewsletterState::Active);
    assert_eq!(NewsletterRole::Admin, NewsletterRole::Admin);
}
