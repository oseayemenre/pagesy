CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS roles (
  id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
  name TEXT UNIQUE NOT NULL
);

CREATE OR REPLACE FUNCTION get_role_id_by_name()
RETURNS UUID
AS $$
  DECLARE role_id UUID;
BEGIN
  SELECT r.id INTO role_id FROM roles WHERE name = 'others';
  RETURN role_id;
END;
$$ language 'plpgsql';

CREATE TABLE IF NOT EXISTS users_roles (
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  role_id UUID DEFAULT get_role_id_by_name(),
  PRIMARY KEY (user_id, role_id),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
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
