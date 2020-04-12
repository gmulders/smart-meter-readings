-- Creates a structure of tables, triggers etc to make insertion and retrieval of time-series data fast and easy.
-- It is heavily inspired on a talk by Steve Simpson.
-- When inserting data on the measurement view, the triggers and functions will make sure that the the data is
-- normalised; the metric data is stored separate from the value data. Besides the benefit of safing space, it also
-- allows us to make an index on the metric id that gives us highly performant select statements. Besides normalisation
-- an insert will also trigger an update on a table that holds aggregated values of the data.

-- Installation:
-- Create a table and make sure that you are allowed to access the database. Then run the following command:
-- `psql -f measurement_ddl.sql <database-name>`

-- Usage:
-- To insert or query data, use the measurement view. To query aggregated data use the aggregate_value_300 table.

CREATE TABLE metric (
    id SERIAL,
    name VARCHAR NOT NULL,
    dimensions JSONB NOT NULL
);

ALTER TABLE metric ADD CONSTRAINT uq_metric_name_dimensions UNIQUE (name, dimensions);
CREATE INDEX ix_metric_name_id ON metric USING BTREE (name, id);
CREATE INDEX ix_metric_dimensions ON metric USING GIN (dimensions);

CREATE TABLE value (
    timestamp TIMESTAMPTZ NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    metric_id INT NOT NULL,
    meta JSON
);

ALTER TABLE value ADD CONSTRAINT uq_value_metric_id_timestamp UNIQUE (metric_id, timestamp);
CREATE INDEX ON value USING BTREE (metric_id, timestamp);

CREATE VIEW measurement AS
SELECT
    timestamp,
    value,
    name,
    dimensions,
    meta
FROM
    value
INNER JOIN metric ON
    metric_id = id;

CREATE FUNCTION create_metric (in_name VARCHAR, in_dimensions JSONB)
RETURNS INT AS
$$
DECLARE
    out_id INT;
BEGIN
    SELECT id INTO out_id
    FROM metric AS m
    WHERE
        m.name = in_name
        AND m.dimensions = in_dimensions;
    IF NOT FOUND THEN
        INSERT INTO metric (name, dimensions)
        VALUES (in_name, in_dimensions)
        RETURNING id INTO out_id;
    END IF;
    RETURN out_id;
END;
$$
LANGUAGE 'plpgsql';

CREATE RULE measurement_insert
AS ON INSERT TO measurement
DO INSTEAD
INSERT INTO value (
    timestamp,
    value,
    metric_id,
    meta
) VALUES (
    NEW.timestamp,
    NEW.value,
    create_metric (
        NEW.name,
        NEW.dimensions),
    NEW.meta
);


CREATE TABLE aggregate_value_300 (
    timestamp TIMESTAMPTZ NOT NULL,
    metric_id INT NOT NULL,
    count DOUBLE PRECISION NOT NULL,
    sum DOUBLE PRECISION NOT NULL,
    sum_squares DOUBLE PRECISION NOT NULL,
    min DOUBLE PRECISION NOT NULL,
    max DOUBLE PRECISION NOT NULL
);

ALTER TABLE aggregate_value_300 ADD CONSTRAINT uq_aggregate_value_300_metric_id_timestamp UNIQUE (metric_id, timestamp);

CREATE FUNCTION time_round(timestamp_in TIMESTAMPTZ, number int)
RETURNS TIMESTAMPTZ AS
$$
DECLARE
    timestamp_out TIMESTAMPTZ;
BEGIN
	SELECT ('epoch'::TIMESTAMP + '1 second'::INTERVAL * (number * floor(extract(EPOCH FROM timestamp_in) / number))) AT TIME ZONE 'utc' INTO timestamp_out;
    RETURN timestamp_out;
END;
$$
LANGUAGE 'plpgsql';


CREATE FUNCTION summarise_300()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO aggregate_value_300 VALUES (
        time_round(NEW.timestamp, 300),
        NEW.metric_id,
        1,
        NEW.value,
        NEW.value * NEW.value,
        NEW.value,
        NEW.value
    )
    ON CONFLICT (metric_id, timestamp)
    DO UPDATE SET
        count = aggregate_value_300.count + EXCLUDED.count,
        sum = aggregate_value_300.sum + EXCLUDED.sum,
        sum_squares = aggregate_value_300.sum_squares + EXCLUDED.sum_squares,
        min = LEAST (aggregate_value_300.min, EXCLUDED.min),
        max = GREATEST (aggregate_value_300.max, EXCLUDED.max);
	RETURN NULL;
END;
$$
LANGUAGE 'plpgsql';


CREATE TRIGGER summarise_300_t
AFTER INSERT ON value
FOR EACH ROW
EXECUTE PROCEDURE summarise_300 ();

