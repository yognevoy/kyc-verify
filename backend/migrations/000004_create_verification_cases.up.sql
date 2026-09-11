CREATE TABLE verification_cases (
    id           UUID PRIMARY KEY,
    applicant_id UUID NOT NULL REFERENCES applicants (id) ON DELETE CASCADE,
    status       TEXT NOT NULL DEFAULT 'draft'
                 CHECK (status IN ('draft', 'submitted', 'in_review', 'approved', 'rejected')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_verification_cases_applicant_id ON verification_cases (applicant_id);
