-- CreateTable
CREATE TABLE "profiles" (
    "id" UUID NOT NULL,
    "user_id" UUID NOT NULL,
    "display_name" VARCHAR(100) NOT NULL,
    "email" VARCHAR(255),
    "phone" VARCHAR(20),
    "avatar_url" VARCHAR(500),
    "cover_url" VARCHAR(500),
    "bio" VARCHAR(1000),
    "date_of_birth" DATE,
    "gender" VARCHAR(20),
    "country_code" VARCHAR(10),
    "city" VARCHAR(100),
    "preferences" JSONB DEFAULT '{}',
    "membership_tier" VARCHAR(20) NOT NULL DEFAULT 'BRONZE',
    "loyalty_points" INTEGER NOT NULL DEFAULT 0,
    "membership_expires_at" TIMESTAMPTZ,
    "is_phone_verified" BOOLEAN NOT NULL DEFAULT false,
    "is_email_verified" BOOLEAN NOT NULL DEFAULT false,
    "is_identity_verified" BOOLEAN NOT NULL DEFAULT false,
    "identity_document" JSONB,
    "following_shop_ids" TEXT[] DEFAULT ARRAY[]::TEXT[],
    "followers_count" INTEGER NOT NULL DEFAULT 0,
    "stats" JSONB DEFAULT '{}',
    "metadata" JSONB DEFAULT '{}',
    "last_active_at" TIMESTAMPTZ,
    "is_deleted" BOOLEAN NOT NULL DEFAULT false,
    "deleted_at" TIMESTAMPTZ,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ NOT NULL,

    CONSTRAINT "profiles_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "shop_configs" (
    "id" UUID NOT NULL,
    "profile_id" UUID NOT NULL,
    "shop_id" VARCHAR(50),
    "shop_name" VARCHAR(100),
    "shop_slug" VARCHAR(100),
    "description" TEXT,
    "logo_url" VARCHAR(500),
    "banner_url" VARCHAR(500),
    "pickup_address_id" UUID,
    "business_type" VARCHAR(20) NOT NULL DEFAULT 'INDIVIDUAL',
    "business_license" TEXT,
    "tax_id" TEXT,
    "bank_account" JSONB,
    "policies" JSONB,
    "operating_hours" JSONB,
    "social_links" JSONB,
    "verification_status" VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    "verified_at" TIMESTAMPTZ,
    "rating" DOUBLE PRECISION NOT NULL DEFAULT 0,
    "total_reviews" INTEGER NOT NULL DEFAULT 0,
    "total_products" INTEGER NOT NULL DEFAULT 0,
    "total_orders" INTEGER NOT NULL DEFAULT 0,
    "total_followers" INTEGER NOT NULL DEFAULT 0,
    "joined_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "shop_configs_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "addresses" (
    "id" UUID NOT NULL,
    "profile_id" UUID NOT NULL,
    "contact_name" VARCHAR(100) NOT NULL,
    "phone" VARCHAR(20) NOT NULL,
    "email" VARCHAR(255),
    "country_code" VARCHAR(10) NOT NULL,
    "province_code" VARCHAR(20) NOT NULL,
    "province_name" TEXT,
    "district_code" VARCHAR(20) NOT NULL,
    "district_name" TEXT,
    "ward_code" VARCHAR(20) NOT NULL,
    "ward_name" TEXT,
    "street_address" VARCHAR(500) NOT NULL,
    "apartment" VARCHAR(100),
    "postal_code" VARCHAR(20),
    "full_address" VARCHAR(1000),
    "coordinates" JSONB,
    "type" TEXT NOT NULL DEFAULT 'HOME',
    "label" VARCHAR(50),
    "is_default" BOOLEAN NOT NULL DEFAULT false,
    "is_default_billing" BOOLEAN NOT NULL DEFAULT false,
    "is_default_pickup" BOOLEAN NOT NULL DEFAULT false,
    "delivery_instructions" VARCHAR(500),
    "is_verified" BOOLEAN NOT NULL DEFAULT false,
    "verified_at" TIMESTAMPTZ,
    "is_deleted" BOOLEAN NOT NULL DEFAULT false,
    "deleted_at" TIMESTAMPTZ,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ NOT NULL,

    CONSTRAINT "addresses_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "profiles_user_id_key" ON "profiles"("user_id");

-- CreateIndex
CREATE INDEX "profiles_user_id_idx" ON "profiles"("user_id");

-- CreateIndex
CREATE INDEX "profiles_email_idx" ON "profiles"("email");

-- CreateIndex
CREATE INDEX "profiles_phone_idx" ON "profiles"("phone");

-- CreateIndex
CREATE UNIQUE INDEX "shop_configs_profile_id_key" ON "shop_configs"("profile_id");

-- CreateIndex
CREATE UNIQUE INDEX "shop_configs_shop_slug_key" ON "shop_configs"("shop_slug");

-- CreateIndex
CREATE INDEX "addresses_profile_id_idx" ON "addresses"("profile_id");

-- AddForeignKey
ALTER TABLE "shop_configs" ADD CONSTRAINT "shop_configs_profile_id_fkey" FOREIGN KEY ("profile_id") REFERENCES "profiles"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "addresses" ADD CONSTRAINT "addresses_profile_id_fkey" FOREIGN KEY ("profile_id") REFERENCES "profiles"("id") ON DELETE CASCADE ON UPDATE CASCADE;
