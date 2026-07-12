CREATE TABLE tasks (
    id           INTEGER PRIMARY KEY,
    parent_id    INTEGER REFERENCES tasks(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    project      TEXT NOT NULL DEFAULT '',
    priority     TEXT NOT NULL DEFAULT ''        CHECK (priority IN ('','L','M','H')),
    status       TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','done','deleted')),
    due          TEXT,
    wait         TEXT,
    scheduled    TEXT,
    recur        TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL,
    modified_at  TEXT NOT NULL,
    completed_at TEXT
);

CREATE TABLE tags (
    task_id  INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    tag      TEXT NOT NULL,
    PRIMARY KEY (task_id, tag)
);

CREATE TABLE notes (
    id         INTEGER PRIMARY KEY,
    task_id    INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    body       TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE activity (
    id       INTEGER PRIMARY KEY,
    op_id    TEXT NOT NULL,
    task_id  INTEGER NOT NULL,
    ts       TEXT NOT NULL,
    kind     TEXT NOT NULL,
    field    TEXT NOT NULL DEFAULT '',
    old_val  TEXT NOT NULL DEFAULT '',
    new_val  TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_tasks_status  ON tasks(status);
CREATE INDEX idx_tasks_project ON tasks(project);
CREATE INDEX idx_tasks_parent  ON tasks(parent_id);
CREATE INDEX idx_tasks_due     ON tasks(due) WHERE due IS NOT NULL;
CREATE INDEX idx_tags_tag      ON tags(tag);
CREATE INDEX idx_activity_op   ON activity(op_id);
CREATE INDEX idx_activity_task ON activity(task_id, ts);
