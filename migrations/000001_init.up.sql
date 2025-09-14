CREATE TABLE IF NOT EXISTS orders (
                                      order_uid        UUID        PRIMARY KEY,
                                      track_number     VARCHAR(32) NOT NULL,
    entry            VARCHAR(16),
    locale           VARCHAR(8),
    internal_signature TEXT,
    customer_id      VARCHAR(64),
    delivery_service VARCHAR(32),
    shardkey         SMALLINT,
    sm_id            INTEGER,
    date_created     TIMESTAMPTZ NOT NULL DEFAULT now()
    );

CREATE TABLE IF NOT EXISTS deliveries (
                                          order_uid UUID PRIMARY KEY
                                          REFERENCES orders(order_uid) ON DELETE CASCADE,
    del_name  TEXT,
    phone     TEXT,
    zip       VARCHAR(16),
    city      TEXT,
    address   TEXT,
    region    TEXT,
    email     TEXT
    );

CREATE TABLE IF NOT EXISTS payments (
                                        order_uid    UUID PRIMARY KEY
                                        REFERENCES orders(order_uid) ON DELETE CASCADE,
    transaction_id  VARCHAR(64),
    request_id   VARCHAR(64),
    currency     CHAR(3),
    provider     VARCHAR(32),
    amount       NUMERIC(12,2),
    payment_dt   BIGINT,
    bank         VARCHAR(64),
    delivery_cost NUMERIC(12,2),
    goods_total   NUMERIC(12,2),
    custom_fee    NUMERIC(12,2)
    );

CREATE TABLE IF NOT EXISTS items (
                                     item_id      BIGSERIAL PRIMARY KEY,
                                     order_uid    UUID NOT NULL
                                     REFERENCES orders(order_uid) ON DELETE CASCADE,
    chrt_id      BIGINT,
    track_number VARCHAR(32),
    price        NUMERIC(12,2),
    rid          VARCHAR(64),
    item_name         TEXT,
    sale         INTEGER,
    item_size    VARCHAR(16),
    total_price  NUMERIC(12,2),
    nm_id        BIGINT,
    brand        TEXT,
    status       INTEGER
    );

CREATE INDEX IF NOT EXISTS idx_orders_track ON orders(track_number);
CREATE INDEX IF NOT EXISTS idx_items_rid   ON items(rid);