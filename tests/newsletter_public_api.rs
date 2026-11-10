use wa_rs::NewsletterPollVote as RootNewsletterPollVote;
use wa_rs::features::NewsletterPollVote as FeatureNewsletterPollVote;

#[test]
fn newsletter_poll_vote_is_public_at_both_import_paths() {
    let _: Option<RootNewsletterPollVote> = None;
    let _: Option<FeatureNewsletterPollVote> = None;
}
