-- Storage nodes: scoped per player+star+(optional planet)
CREATE TABLE econ2_nodes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id  UUID NOT NULL,
    star_id    UUID NOT NULL,
    planet_id  UUID,
    level      TEXT NOT NULL, -- 'planetary' | 'orbital' | 'intersystem'
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX econ2_nodes_orbital  ON econ2_nodes (player_id, star_id) WHERE planet_id IS NULL;
CREATE UNIQUE INDEX econ2_nodes_planet   ON econ2_nodes (player_id, star_id, planet_id) WHERE planet_id IS NOT NULL;

-- Item stock with allocation tracking
CREATE TABLE econ2_item_stock (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id    UUID NOT NULL REFERENCES econ2_nodes(id) ON DELETE CASCADE,
    item_id    TEXT NOT NULL,
    total      FLOAT NOT NULL DEFAULT 0 CHECK (total >= 0),
    allocated  FLOAT NOT NULL DEFAULT 0 CHECK (allocated >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (node_id, item_id)
);

-- Standalone facilities (not linked to old economy tables)
CREATE TABLE econ2_facilities (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id        UUID NOT NULL,
    star_id          UUID NOT NULL,
    planet_id        UUID,
    node_id          UUID NOT NULL REFERENCES econ2_nodes(id),
    factory_type     TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'idle',
    config           JSONB NOT NULL DEFAULT '{}',
    current_order_id UUID,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Production orders with snapshot inputs
CREATE TABLE econ2_orders (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id        UUID NOT NULL,
    star_id          UUID NOT NULL,
    node_id          UUID NOT NULL REFERENCES econ2_nodes(id),
    facility_id      UUID REFERENCES econ2_facilities(id),
    order_type       TEXT NOT NULL, -- 'batch' | 'continuous'
    status           TEXT NOT NULL DEFAULT 'pending',
    recipe_id        TEXT NOT NULL,
    product_id       TEXT NOT NULL,
    factory_type     TEXT NOT NULL,
    inputs           JSONB NOT NULL DEFAULT '[]',
    base_yield       FLOAT NOT NULL DEFAULT 1,
    recipe_ticks     INT NOT NULL DEFAULT 1,
    efficiency       FLOAT NOT NULL DEFAULT 1,
    target_qty       FLOAT NOT NULL DEFAULT 0,
    allocated_inputs JSONB NOT NULL DEFAULT '{}',
    produced_qty     FLOAT NOT NULL DEFAULT 0,
    priority         INT NOT NULL DEFAULT 5,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX econ2_orders_node_type ON econ2_orders (node_id, factory_type, status);

-- Transport routes between nodes
CREATE TABLE econ2_routes (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id            UUID NOT NULL,
    from_node_id         UUID NOT NULL REFERENCES econ2_nodes(id),
    to_node_id           UUID NOT NULL REFERENCES econ2_nodes(id),
    capacity_per_tick    FLOAT NOT NULL DEFAULT 0,
    min_continuous_share FLOAT NOT NULL DEFAULT 0.20,
    status               TEXT NOT NULL DEFAULT 'active',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Ships on routes
CREATE TABLE econ2_ships (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    route_id   UUID NOT NULL REFERENCES econ2_routes(id) ON DELETE CASCADE,
    state      TEXT NOT NULL DEFAULT 'loading',
    cargo      JSONB NOT NULL DEFAULT '{}',
    cargo_max  FLOAT NOT NULL,
    eta_tick   BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- System warnings
CREATE TABLE econ2_warnings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id   UUID NOT NULL,
    order_id    UUID REFERENCES econ2_orders(id),
    route_id    UUID REFERENCES econ2_routes(id),
    type        TEXT NOT NULL,
    message     TEXT NOT NULL,
    tick_n      BIGINT NOT NULL,
    resolved_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
