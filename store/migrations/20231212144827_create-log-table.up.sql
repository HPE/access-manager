/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

CREATE TABLE log
(
    id          BIGSERIAL PRIMARY KEY,
    principal    TEXT,
    dataset      TEXT,
    global       BIGINT,
    operation    INT,
    requested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expired_at   TIMESTAMP
);


CREATE INDEX idx_dataset ON log(dataset);
CREATE INDEX idx_principal ON log(principal);
CREATE INDEX idx_operation ON log(operation);
CREATE INDEX idx_dataset_principal ON log(dataset, principal);
CREATE INDEX idx_requested_expired ON log(requested_at, expired_at);
CREATE INDEX idx_dataset_principal_operation ON log(dataset, principal, operation);
