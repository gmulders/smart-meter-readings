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
-- To insert or query data, use the measurement view. To query aggregated data use the aggregate_value_* tables.

CREATE TABLE metric (
    id SERIAL,
    name VARCHAR NOT NULL,
    dimensions JSONB NOT NULL
);

ALTER TABLE metric ADD CONSTRAINT pk_metric PRIMARY KEY (id);
ALTER TABLE metric ADD CONSTRAINT uq_metric_id_name_dimensions UNIQUE (id, name, dimensions);
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
ALTER TABLE value ADD CONSTRAINT fk_value_metric FOREIGN KEY (metric_id) REFERENCES metric (id);
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

CREATE FUNCTION time_floor(timestamp_in TIMESTAMPTZ, number int)
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


-- 5 minutes
CREATE TABLE aggregate_value_5m (
    timestamp TIMESTAMPTZ NOT NULL,
    metric_id INT NOT NULL,
    count DOUBLE PRECISION NOT NULL,
    sum DOUBLE PRECISION NOT NULL,
    sum_squares DOUBLE PRECISION NOT NULL,
    min DOUBLE PRECISION NOT NULL,
    max DOUBLE PRECISION NOT NULL
);

ALTER TABLE aggregate_value_5m ADD CONSTRAINT fk_aggregate_value_5m_metric FOREIGN KEY (metric_id) REFERENCES metric (id);
ALTER TABLE aggregate_value_5m ADD CONSTRAINT uq_aggregate_value_5m_metric_id_timestamp UNIQUE (metric_id, timestamp);
CREATE INDEX ON aggregate_value_5m USING BTREE (metric_id, timestamp);


CREATE FUNCTION summarise_5m()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO aggregate_value_5m VALUES (
        time_floor(NEW.timestamp, 300),
        NEW.metric_id,
        1,
        NEW.value,
        NEW.value * NEW.value,
        NEW.value,
        NEW.value
    )
    ON CONFLICT (metric_id, timestamp)
    DO UPDATE SET
        count = aggregate_value_5m.count + EXCLUDED.count,
        sum = aggregate_value_5m.sum + EXCLUDED.sum,
        sum_squares = aggregate_value_5m.sum_squares + EXCLUDED.sum_squares,
        min = LEAST (aggregate_value_5m.min, EXCLUDED.min),
        max = GREATEST (aggregate_value_5m.max, EXCLUDED.max);
	RETURN NULL;
END;
$$
LANGUAGE 'plpgsql';


CREATE TRIGGER summarise_5m_t
AFTER INSERT ON value
FOR EACH ROW
EXECUTE PROCEDURE summarise_5m();


-- 30 minutes
CREATE TABLE aggregate_value_30m (
    timestamp TIMESTAMPTZ NOT NULL,
    metric_id INT NOT NULL,
    count DOUBLE PRECISION NOT NULL,
    sum DOUBLE PRECISION NOT NULL,
    sum_squares DOUBLE PRECISION NOT NULL,
    min DOUBLE PRECISION NOT NULL,
    max DOUBLE PRECISION NOT NULL
);

ALTER TABLE aggregate_value_30m ADD CONSTRAINT fk_aggregate_value_30m_metric FOREIGN KEY (metric_id) REFERENCES metric (id);
ALTER TABLE aggregate_value_30m ADD CONSTRAINT uq_aggregate_value_30m_metric_id_timestamp UNIQUE (metric_id, timestamp);
CREATE INDEX ON aggregate_value_30m USING BTREE (metric_id, timestamp);


CREATE FUNCTION summarise_30m()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO aggregate_value_30m VALUES (
        time_floor(NEW.timestamp, 1800),
        NEW.metric_id,
        1,
        NEW.value,
        NEW.value * NEW.value,
        NEW.value,
        NEW.value
    )
    ON CONFLICT (metric_id, timestamp)
    DO UPDATE SET
        count = aggregate_value_30m.count + EXCLUDED.count,
        sum = aggregate_value_30m.sum + EXCLUDED.sum,
        sum_squares = aggregate_value_30m.sum_squares + EXCLUDED.sum_squares,
        min = LEAST (aggregate_value_30m.min, EXCLUDED.min),
        max = GREATEST (aggregate_value_30m.max, EXCLUDED.max);
	RETURN NULL;
END;
$$
LANGUAGE 'plpgsql';


CREATE TRIGGER summarise_30m_t
AFTER INSERT ON value
FOR EACH ROW
EXECUTE PROCEDURE summarise_30m();


-- 3 hours
CREATE TABLE aggregate_value_3h (
    timestamp TIMESTAMPTZ NOT NULL,
    metric_id INT NOT NULL,
    count DOUBLE PRECISION NOT NULL,ƒ
    sum DOUBLE PRECISION NOT NULL,
    sum_squares DOUBLE PRECISION NOT NULL,
    min DOUBLE PRECISION NOT NULL,
    max DOUBLE PRECISION NOT NULL
);

ALTER TABLE aggregate_value_3h ADD CONSTRAINT fk_aggregate_value_3h_metric FOREIGN KEY (metric_id) REFERENCES metric (id);
ALTER TABLE aggregate_value_3h ADD CONSTRAINT uq_aggregate_value_3h_metric_id_timestamp UNIQUE (metric_id, timestamp);
CREATE INDEX ON aggregate_value_3h USING BTREE (metric_id, timestamp);


CREATE FUNCTION summarise_3h()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO aggregate_value_3h VALUES (
        time_floor(NEW.timestamp, 10800),
        NEW.metric_id,
        1,
        NEW.value,
        NEW.value * NEW.value,
        NEW.value,
        NEW.value
    )
    ON CONFLICT (metric_id, timestamp)
    DO UPDATE SET
        count = aggregate_value_3h.count + EXCLUDED.count,
        sum = aggregate_value_3h.sum + EXCLUDED.sum,
        sum_squares = aggregate_value_3h.sum_squares + EXCLUDED.sum_squares,
        min = LEAST (aggregate_value_3h.min, EXCLUDED.min),
        max = GREATEST (aggregate_value_3h.max, EXCLUDED.max);
	RETURN NULL;
END;
$$
LANGUAGE 'plpgsql';


CREATE TRIGGER summarise_3h_t
AFTER INSERT ON value
FOR EACH ROW
EXECUTE PROCEDURE summarise_3h();




-- 1 day
CREATE TABLE aggregate_value_1d (
    timestamp TIMESTAMPTZ NOT NULL,
    metric_id INT NOT NULL,
    count DOUBLE PRECISION NOT NULL,ƒ
    sum DOUBLE PRECISION NOT NULL,
    sum_squares DOUBLE PRECISION NOT NULL,
    min DOUBLE PRECISION NOT NULL,
    max DOUBLE PRECISION NOT NULL
);

ALTER TABLE aggregate_value_1d ADD CONSTRAINT fk_aggregate_value_1d_metric FOREIGN KEY (metric_id) REFERENCES metric (id);
ALTER TABLE aggregate_value_1d ADD CONSTRAINT uq_aggregate_value_1d_metric_id_timestamp UNIQUE (metric_id, timestamp);
CREATE INDEX ON aggregate_value_1d USING BTREE (metric_id, timestamp);


CREATE FUNCTION summarise_1d()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO aggregate_value_1d VALUES (
        time_floor(NEW.timestamp, 86400),
        NEW.metric_id,
        1,
        NEW.value,
        NEW.value * NEW.value,
        NEW.value,
        NEW.value
    )
    ON CONFLICT (metric_id, timestamp)
    DO UPDATE SET
        count = aggregate_value_1d.count + EXCLUDED.count,
        sum = aggregate_value_1d.sum + EXCLUDED.sum,
        sum_squares = aggregate_value_1d.sum_squares + EXCLUDED.sum_squares,
        min = LEAST (aggregate_value_1d.min, EXCLUDED.min),
        max = GREATEST (aggregate_value_1d.max, EXCLUDED.max);
	RETURN NULL;
END;
$$
LANGUAGE 'plpgsql';


CREATE TRIGGER summarise_1d_t
AFTER INSERT ON value
FOR EACH ROW
EXECUTE PROCEDURE summarise_1d();






-- 1 week
CREATE TABLE aggregate_value_1w (
    timestamp TIMESTAMPTZ NOT NULL,
    metric_id INT NOT NULL,
    count DOUBLE PRECISION NOT NULL,ƒ
    sum DOUBLE PRECISION NOT NULL,
    sum_squares DOUBLE PRECISION NOT NULL,
    min DOUBLE PRECISION NOT NULL,
    max DOUBLE PRECISION NOT NULL
);

ALTER TABLE aggregate_value_1w ADD CONSTRAINT fk_aggregate_value_1w_metric FOREIGN KEY (metric_id) REFERENCES metric (id);
ALTER TABLE aggregate_value_1w ADD CONSTRAINT uq_aggregate_value_1w_metric_id_timestamp UNIQUE (metric_id, timestamp);
CREATE INDEX ON aggregate_value_1w USING BTREE (metric_id, timestamp);


CREATE FUNCTION summarise_1w()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO aggregate_value_1w VALUES (
        time_floor(NEW.timestamp, 604800),
        NEW.metric_id,
        1,
        NEW.value,
        NEW.value * NEW.value,
        NEW.value,
        NEW.value
    )
    ON CONFLICT (metric_id, timestamp)
    DO UPDATE SET
        count = aggregate_value_1w.count + EXCLUDED.count,
        sum = aggregate_value_1w.sum + EXCLUDED.sum,
        sum_squares = aggregate_value_1w.sum_squares + EXCLUDED.sum_squares,
        min = LEAST (aggregate_value_1w.min, EXCLUDED.min),
        max = GREATEST (aggregate_value_1w.max, EXCLUDED.max);
	RETURN NULL;
END;
$$
LANGUAGE 'plpgsql';


CREATE TRIGGER summarise_1w_t
AFTER INSERT ON value
FOR EACH ROW
EXECUTE PROCEDURE summarise_1w();

