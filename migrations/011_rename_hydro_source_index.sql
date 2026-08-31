-- +goose Up
-- +goose StatementBegin
ALTER TABLE hydro_level_readings
    RENAME COLUMN change_cm_per_hour TO source_hdi_ihr;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE hydro_level_readings
    RENAME COLUMN source_hdi_ihr TO change_cm_per_hour;
-- +goose StatementEnd
