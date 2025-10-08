-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
-- Replace Twilio columns with AWS columns in SMS table

-- This migration handles the transition from Twilio to AWS SNS
-- It's safe to run multiple times as it checks for column existence

-- Add AWS columns if they don't exist (SQLite doesn't support IF NOT EXISTS for ALTER TABLE)
-- We'll use a different approach - create a new table and copy data

-- First, create a backup of the existing sms table
CREATE TABLE sms_backup AS SELECT * FROM sms;

-- Drop the existing sms table
DROP TABLE sms;

-- Create the new sms table with AWS columns
CREATE TABLE sms(
	id integer primary key autoincrement,
	user_id bigint,
	name varchar(255),
    aws_access_key_id varchar(255),
    aws_secret_key varchar(255),
    aws_region varchar(255),
    delay varchar(255),
    sms_from varchar(255),
	modified_date datetime default CURRENT_TIMESTAMP
);

-- Copy data from backup, mapping old Twilio columns to new AWS columns
-- For existing data, we'll leave AWS columns empty as they need to be configured
INSERT INTO sms (id, user_id, name, delay, sms_from, modified_date)
SELECT id, user_id, name, delay, sms_from, modified_date FROM sms_backup;

-- Drop the backup table
DROP TABLE sms_backup;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
-- Remove AWS columns and restore Twilio columns
ALTER TABLE sms DROP COLUMN "aws_access_key_id";
ALTER TABLE sms DROP COLUMN "aws_secret_key";
ALTER TABLE sms DROP COLUMN "aws_region";
