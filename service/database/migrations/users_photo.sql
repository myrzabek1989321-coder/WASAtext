
-- add profile photo to users

ALTER TABLE users
ADD COLUMN photo_mime_type TEXT;

ALTER TABLE users
ADD COLUMN photo_data BLOB;
