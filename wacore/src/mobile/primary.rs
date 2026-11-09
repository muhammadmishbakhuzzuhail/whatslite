//! Primary-side companion approval primitives.
//!
//! Phase 1 provides the account/reference-bound state machine and validation
//! seams. ADV-signed wire construction is covered by the ADV module.

use subtle::ConstantTimeEq;
use wacore_binary::Jid;
use wacore_binary::jid::JidExt;

#[derive(Debug, thiserror::Error, PartialEq, Eq)]
#[non_exhaustive]
pub enum PrimaryLinkError {
    #[error("invalid primary linking input: {0}")]
    InvalidInput(&'static str),
    #[error("primary linking cryptographic authentication failed")]
    Authentication,
}

/// Signed material issued for a companion device.
#[derive(Debug, Clone)]
pub struct SignedCompanionIdentity {
    pub device_identity: Vec<u8>,
    pub key_index_list: Vec<u8>,
}

fn validate_accounts(accounts: &[Jid]) -> Result<(), PrimaryLinkError> {
    if accounts.is_empty()
        || accounts.len() > 2
        || accounts.iter().any(|jid| {
            jid.user.is_empty()
                || jid.device != 0
                || jid.agent != 0
                || jid.integrator != 0
                || !(jid.is_pn() || jid.is_lid())
        })
        || accounts.len() == 2 && accounts[0].server == accounts[1].server
    {
        return Err(PrimaryLinkError::InvalidInput("primary account JIDs"));
    }
    Ok(())
}

fn validate_companion(jid: &Jid, accounts: &[Jid]) -> Result<(), PrimaryLinkError> {
    validate_accounts(accounts)?;
    if jid.device == 0
        || jid.agent != 0
        || jid.integrator != 0
        || jid.is_hosted()
        || !(jid.is_pn() || jid.is_lid())
        || !accounts
            .iter()
            .any(|account| account.user == jid.user && account.server == jid.server)
    {
        return Err(PrimaryLinkError::Authentication);
    }
    Ok(())
}

/// Account- and reference-bound pending approval.
#[derive(Debug)]
pub struct PendingCompanion {
    reference: Vec<u8>,
    own_accounts: Vec<Jid>,
    peer_identity: [u8; 32],
    issued: SignedCompanionIdentity,
}

impl PendingCompanion {
    pub fn new(
        reference: &[u8],
        own_accounts: &[Jid],
        peer_identity: &[u8; 32],
        issued: SignedCompanionIdentity,
    ) -> Result<Self, PrimaryLinkError> {
        validate_accounts(own_accounts)?;
        if issued.device_identity.is_empty()
            || issued.device_identity.len() > 16384
            || issued.key_index_list.is_empty()
            || issued.key_index_list.len() > 16384
        {
            return Err(PrimaryLinkError::InvalidInput("signed companion identity"));
        }
        if reference.is_empty() || reference.len() > 4096 {
            return Err(PrimaryLinkError::InvalidInput("pairing reference"));
        }
        Ok(Self {
            reference: reference.to_vec(),
            own_accounts: own_accounts.to_vec(),
            peer_identity: *peer_identity,
            issued,
        })
    }

    pub fn acknowledge(self, device_jid: &Jid) -> Result<AcknowledgedCompanion, PrimaryLinkError> {
        validate_companion(device_jid, &self.own_accounts)?;
        Ok(AcknowledgedCompanion {
            pending: self,
            device_jid: device_jid.clone(),
        })
    }
}

/// Server acknowledgement alone cannot produce a proof-verified device.
#[derive(Debug)]
pub struct AcknowledgedCompanion {
    pending: PendingCompanion,
    device_jid: Jid,
}

impl AcknowledgedCompanion {
    pub fn verify(
        &self,
        reference: &[u8],
        device_jid: &Jid,
        proof: &[u8],
    ) -> Result<VerifiedCompanion, PrimaryLinkError> {
        if !bool::from(self.pending.reference.as_slice().ct_eq(reference))
            || device_jid != &self.device_jid
            || proof.is_empty()
        {
            return Err(PrimaryLinkError::Authentication);
        }
        Ok(VerifiedCompanion {
            device_jid: device_jid.clone(),
            peer_identity: self.pending.peer_identity,
            proof: proof.to_vec(),
            signed_identity: self.pending.issued.clone(),
        })
    }
}

/// Authenticated material awaiting the host's durable publication.
#[derive(Debug)]
pub struct VerifiedCompanion {
    device_jid: Jid,
    peer_identity: [u8; 32],
    proof: Vec<u8>,
    signed_identity: SignedCompanionIdentity,
}

impl VerifiedCompanion {
    pub fn device_jid(&self) -> &Jid {
        &self.device_jid
    }
    pub fn peer_identity(&self) -> &[u8; 32] {
        &self.peer_identity
    }
    pub fn proof(&self) -> &[u8] {
        &self.proof
    }
    pub fn signed_identity(&self) -> &SignedCompanionIdentity {
        &self.signed_identity
    }
}
