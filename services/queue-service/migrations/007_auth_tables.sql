CREATE TABLE roles (
  role_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  name TEXT NOT NULL
);

CREATE TABLE users (
  user_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  role_id UUID NOT NULL REFERENCES roles(role_id),
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, email)
);

CREATE TABLE user_branch_access (
  user_id UUID NOT NULL REFERENCES users(user_id),
  branch_id UUID NOT NULL REFERENCES branches(branch_id),
  PRIMARY KEY (user_id, branch_id)
);

CREATE TABLE user_service_access (
  user_id UUID NOT NULL REFERENCES users(user_id),
  service_id UUID NOT NULL REFERENCES services(service_id),
  PRIMARY KEY (user_id, service_id)
);

CREATE TABLE sessions (
  session_id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(user_id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL
);
