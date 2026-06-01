-- Drop all tables in reverse order to avoid foreign key issues

DROP TABLE IF EXISTS oauth_providers;
DROP TABLE IF EXISTS review_comments;
DROP TABLE IF EXISTS branch_protections;
DROP TABLE IF EXISTS releases;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS labels;
DROP TABLE IF EXISTS pull_requests;
DROP TABLE IF EXISTS issues;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS repos;
DROP TABLE IF EXISTS users;
