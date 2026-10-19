-- add profile photo fields to group conversations

ALTER TABLE conversations
ADD COLUMN photo_mime_type TEXT;

ALTER TABLE conversations
ADD COLUMN photo_data BLOB;
