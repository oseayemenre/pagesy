DROP TABLE IF EXISTS roles_privileges;
DROP TABLE IF EXISTS "privileges";
DROP TABLE IF EXISTS users_roles;
DROP TABLE IF EXISTS roles;
DROP TRIGGER IF EXISTS match_user_to_roles_on_create_account ON users;
DROP FUNCTION IF EXISTS match_user_to_roles_on_create_account_func;
