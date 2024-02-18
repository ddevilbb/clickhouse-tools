CREATE TABLE IF NOT EXISTS test_table ON CLUSTER ch_cluster (
    `id` UUID DEFAULT generateUUIDv4() Codec(LZ4),
    sign Int8 Codec(DoubleDelta, LZ4),
    version UInt32 Codec(DoubleDelta, LZ4),
    test_data String Codec(LZ4),
    created_at datetime Codec(DoubleDelta, LZ4)
) Engine = ReplicatedVersionedCollapsingMergeTree('/clickhouse/tables/versioned/{shard}/test_table', '{replica}', sign, version)
PARTITION BY toYYYYMM(`created_at`)
PRIMARY KEY (`id`);

INSERT INTO test_table(sign, version, test_data, created_at) VALUES (1, 1, 'test_data', now())
