-- +goose Up
-- +goose StatementBegin
ALTER TABLE weather_data
    ADD COLUMN wh65batt REAL,       -- Напряжение батареек 2×AA датчика WS90 (вольты)
    ADD COLUMN ws90cap_volt REAL;   -- Напряжение аккумулятора от солнечной панели (вольты)

COMMENT ON COLUMN weather_data.wh65batt IS 'Напряжение батареек 2×AA внешнего датчика (вольты). Норма: 2.4-3.0V, требуется замена при < 2.4V';
COMMENT ON COLUMN weather_data.ws90cap_volt IS 'Напряжение суперконденсатора/аккумулятора, заряжаемого от солнечной панели (вольты). Норма: 3.0-4.3V';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE weather_data
    DROP COLUMN wh65batt,
    DROP COLUMN ws90cap_volt;
-- +goose StatementEnd
