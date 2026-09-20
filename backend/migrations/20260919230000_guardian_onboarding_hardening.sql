-- +goose Up

alter table guardian_invitation_contacts
    add column accepted_consent_id public.xid20;

-- The migrator owns these FORCE RLS tables. Temporarily restore the owner
-- exception so this one-time, cross-tenant backfill can inspect contacts and
-- consents without inventing an app.organization_id for the migration session.
alter table guardian_invitation_contacts no force row level security;
alter table guardian_onboarding_consents no force row level security;

with candidates as (
    select
        contact.id as contact_id,
        consent.id as consent_id,
        count(*) over (partition by contact.id) as contact_candidates,
        count(*) over (partition by consent.id) as consent_candidates
    from guardian_invitation_contacts contact
    join access_tokens invitation on invitation.id = contact.invitation_token_id
    join guardian_onboarding_consents consent
        on consent.organization_id = contact.organization_id
        and consent.school_year_id = contact.school_year_id
        and lower(consent.verified_email) = lower(contact.email)
    where invitation.purpose = 'guardian_invitation'
      and invitation.consumed_at is not null
), unambiguous as (
    select contact_id, consent_id
    from candidates
    where contact_candidates = 1 and consent_candidates = 1
)
update guardian_invitation_contacts contact
set accepted_consent_id = unambiguous.consent_id
from unambiguous
where contact.id = unambiguous.contact_id;

alter table guardian_invitation_contacts
    add constraint guardian_invitation_contacts_accepted_consent_fk
        foreign key (accepted_consent_id, organization_id, school_year_id)
        references guardian_onboarding_consents (id, organization_id, school_year_id)
        on delete set null (accepted_consent_id),
    add constraint guardian_invitation_contacts_accepted_consent_unique
        unique (accepted_consent_id);

alter table guardian_invitation_contacts force row level security;
alter table guardian_onboarding_consents force row level security;

-- +goose Down

alter table guardian_invitation_contacts
    drop constraint if exists guardian_invitation_contacts_accepted_consent_unique,
    drop constraint if exists guardian_invitation_contacts_accepted_consent_fk,
    drop column if exists accepted_consent_id;

