ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS recurrence JSONB DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS parent_id  UUID  DEFAULT NULL REFERENCES tasks(id) ON DELETE SET NULL;

COMMENT ON COLUMN tasks.recurrence IS
    'Optional recurrence rule. JSON object with type and type-specific fields.
     Examples:
       {"type":"daily","interval":1}
       {"type":"monthly","day_of_month":15}
       {"type":"specific_dates","dates":["2024-06-01","2024-07-04"]}
       {"type":"even_odd","parity":"even"}';

COMMENT ON COLUMN tasks.parent_id IS
    'For tasks generated from a recurrence template — reference to the parent (template) task.';
