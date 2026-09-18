-- file_previews was scanned from every upload row but never written to or
-- branched on anywhere in the application — dead column left over from
-- planning, never cleaned up. See PROJECT_DOCUMENTATION.md's rewrite notes.
ALTER TABLE uploads DROP COLUMN file_previews;
