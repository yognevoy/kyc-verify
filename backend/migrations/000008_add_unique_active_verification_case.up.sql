CREATE UNIQUE INDEX idx_verification_cases_one_active_per_applicant
    ON verification_cases (applicant_id)
    WHERE status IN ('draft', 'submitted', 'in_review');
