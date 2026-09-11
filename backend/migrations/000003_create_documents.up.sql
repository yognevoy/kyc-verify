CREATE TABLE documents (
    id           UUID PRIMARY KEY,
    applicant_id UUID NOT NULL REFERENCES applicants (id) ON DELETE CASCADE,
    type         TEXT NOT NULL CHECK (type IN ('passport', 'selfie', 'proof_of_address')),
    file_path    TEXT NOT NULL,
    uploaded_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_documents_applicant_id ON documents (applicant_id);
