CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS roles (
  id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
  name TEXT UNIQUE NOT NULL
);

CREATE OR REPLACE FUNCTION match_user_to_roles_on_create_account_func()
RETURNS TRIGGER
AS $$
DECLARE
  role_id UUID;
BEGIN
  SELECT id INTO role_id FROM roles WHERE roles.name = 'others';

  INSERT INTO users_roles(user_id, role_id)
  VALUES(NEW.id, role_id);

  RETURN NEW;
END; $$ language 'plpgsql';

CREATE TRIGGER match_user_to_roles_on_create_account
AFTER INSERT ON users
FOR EACH ROW
  EXECUTE FUNCTION match_user_to_roles_on_create_account_func();

CREATE TABLE IF NOT EXISTS users_roles (
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS "privileges"(
  id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
  name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS roles_privileges (
  role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  privilege_id UUID NOT NULL REFERENCES "privileges"(id) ON DELETE CASCADE,
  PRIMARY KEY(role_id, privilege_id)
);

INSERT INTO roles (name) VALUES ('admin'), ('others');

INSERT INTO privileges (name) 
VALUES('books:upload'),
      ('creator:books'),
      ('books:stats'),
      ('mark:complete'),
      ('upload:chapters'),
      ('delete:chapters'),
      ('books:approve'),
      ('ban:users'),
      ('recent:reads'),
      ('newly:updated'),
      ('get:recommendations'),
      ('get:books'),
      ('add:library:books'),
      ('get:libary:books'),
      ('remove:libary:books'),
      ('coins'),
      ('books:comment'),
      ('get:books:comment'),
      ('books:delete'),
      ('books:edit');

INSERT INTO roles_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r
JOIN "privileges" p ON TRUE
WHERE r.name = 'admin';

INSERT INTO roles_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r
JOIN "privileges" p ON p.name IN(
  'books:upload',
  'creator:books',
  'books:stats',
  'mark:complete',
  'upload:chapters',
  'delete:chapters',
  'recent:reads',
  'newly:updated'
  'get:recommendations',
  'get:books',
  'add:library:books',
  'get:library:books',
  'remove:library:books',
  'coins',
  'books:comment',
  'get:books:comment',
  'books:delete',
  'books:edit'
)
WHERE r.name = 'others';
