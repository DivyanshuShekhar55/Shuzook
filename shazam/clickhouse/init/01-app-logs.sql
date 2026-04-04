CREATE DATABASE IF NOT EXISTS observability;

CREATE TABLE IF NOT EXISTS observability.app_logs
(
    time String,
    level String,
    msg String,
    caller String,
    git_commit String,
    trace_id String,
    span_id String,
    source String,
    route String,
    method String,
    env String
)
ENGINE = MergeTree
ORDER BY tuple();
