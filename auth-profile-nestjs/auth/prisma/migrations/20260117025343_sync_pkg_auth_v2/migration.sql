-- CreateTable
CREATE TABLE "authorization_policies" (
    "id" VARCHAR(100) NOT NULL,
    "type" VARCHAR(50) NOT NULL,
    "subject" VARCHAR(200) NOT NULL,
    "object" VARCHAR(200) NOT NULL,
    "action" VARCHAR(100) NOT NULL,
    "effect" VARCHAR(20) NOT NULL DEFAULT 'allow',
    "conditions" JSONB,
    "priority" INTEGER NOT NULL DEFAULT 0,
    "domain" VARCHAR(100),
    "tenant_id" VARCHAR(100),
    "description" VARCHAR(500),
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ NOT NULL,

    CONSTRAINT "authorization_policies_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "authorization_roles" (
    "id" VARCHAR(100) NOT NULL,
    "name" VARCHAR(200) NOT NULL,
    "description" VARCHAR(500),
    "parent_id" VARCHAR(100),
    "metadata" JSONB,
    "domain" VARCHAR(100),
    "tenant_id" VARCHAR(100),
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "is_system" BOOLEAN NOT NULL DEFAULT false,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ NOT NULL,

    CONSTRAINT "authorization_roles_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "authorization_permissions" (
    "id" VARCHAR(100) NOT NULL,
    "resource" VARCHAR(200) NOT NULL,
    "action" VARCHAR(100) NOT NULL,
    "scope" VARCHAR(50) NOT NULL DEFAULT 'global',
    "conditions" JSONB,
    "description" VARCHAR(500),
    "domain" VARCHAR(100),
    "tenant_id" VARCHAR(100),
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ NOT NULL,

    CONSTRAINT "authorization_permissions_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "authorization_role_permissions" (
    "role_id" VARCHAR(100) NOT NULL,
    "permission_id" VARCHAR(100) NOT NULL,

    CONSTRAINT "authorization_role_permissions_pkey" PRIMARY KEY ("role_id","permission_id")
);

-- CreateTable
CREATE TABLE "authorization_user_roles" (
    "id" VARCHAR(100) NOT NULL,
    "user_id" UUID NOT NULL,
    "role_id" VARCHAR(100) NOT NULL,
    "domain" VARCHAR(100),
    "tenant_id" VARCHAR(100),
    "expires_at" TIMESTAMPTZ,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ NOT NULL,

    CONSTRAINT "authorization_user_roles_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "authorization_policies_domain_idx" ON "authorization_policies"("domain");

-- CreateIndex
CREATE INDEX "authorization_policies_tenant_id_idx" ON "authorization_policies"("tenant_id");

-- CreateIndex
CREATE INDEX "authorization_roles_parent_id_idx" ON "authorization_roles"("parent_id");

-- CreateIndex
CREATE INDEX "authorization_roles_domain_idx" ON "authorization_roles"("domain");

-- CreateIndex
CREATE INDEX "authorization_roles_tenant_id_idx" ON "authorization_roles"("tenant_id");

-- CreateIndex
CREATE INDEX "authorization_permissions_domain_idx" ON "authorization_permissions"("domain");

-- CreateIndex
CREATE INDEX "authorization_permissions_tenant_id_idx" ON "authorization_permissions"("tenant_id");

-- CreateIndex
CREATE INDEX "authorization_user_roles_user_id_idx" ON "authorization_user_roles"("user_id");

-- CreateIndex
CREATE INDEX "authorization_user_roles_role_id_idx" ON "authorization_user_roles"("role_id");

-- CreateIndex
CREATE INDEX "authorization_user_roles_domain_idx" ON "authorization_user_roles"("domain");

-- CreateIndex
CREATE INDEX "authorization_user_roles_tenant_id_idx" ON "authorization_user_roles"("tenant_id");

-- AddForeignKey
ALTER TABLE "authorization_roles" ADD CONSTRAINT "authorization_roles_parent_id_fkey" FOREIGN KEY ("parent_id") REFERENCES "authorization_roles"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "authorization_role_permissions" ADD CONSTRAINT "authorization_role_permissions_role_id_fkey" FOREIGN KEY ("role_id") REFERENCES "authorization_roles"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "authorization_role_permissions" ADD CONSTRAINT "authorization_role_permissions_permission_id_fkey" FOREIGN KEY ("permission_id") REFERENCES "authorization_permissions"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "authorization_user_roles" ADD CONSTRAINT "authorization_user_roles_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "authorization_user_roles" ADD CONSTRAINT "authorization_user_roles_role_id_fkey" FOREIGN KEY ("role_id") REFERENCES "authorization_roles"("id") ON DELETE CASCADE ON UPDATE CASCADE;
