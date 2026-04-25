ALTER TABLE tasks
    DROP COLUMN IF EXISTS recurrence,
    DROP COLUMN IF EXISTS parent_id;
