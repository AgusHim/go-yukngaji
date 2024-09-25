CREATE TABLE "users" (
  "id" varchar PRIMARY KEY,
  "name" varchar NOT NULL,
  "username" varchar DEFAULT ('anonim'),
  "gender" varchar NOT NULL DEFAULT ('male'),
  "age" integer NOT NULL,
  "birth_date" timestamp,
  "phone" varchar,
  "email" varchar,
  "instagram" varchar,
  "password" varchar,
  "address" varchar,
  "google_id" varchar,
  "image_url" varchar,
  "role" varchar DEFAULT ('jamaah'),
  "activity" varchar DEFAULT ('pelajar'),
  "province_code" varchar,
  "district_code" varchar,
  "sub_district_code" varchar,
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

CREATE TABLE "divisi" (
  "id" varchar PRIMARY KEY,
  "name" varchar NOT NULL,
  "regional" varchar NOT NULL,
  "created_at" timestamp,
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

CREATE TABLE "events" (
  "id" varchar PRIMARY KEY,
  "slug" varchar NOT NULL,
  "code" varchar NOT NULL,
  "title" varchar NOT NULL,
  "desc" text NOT NULL,
  "image_url" varchar NOT NULL,
  "speaker" varchar NOT NULL,
  "divisi_id" varchar NOT NULL,
  "start_at" timestamp NOT NULL,
  "end_at" timestamp NOT NULL,
  "participant" integer DEFAULT (0),
  "close_at" timestamp,
  "isPublished" boolean DEFAULT (true),
  "isWhitelistOnly" boolean DEFAULT (false),
  "allowed_gender" varchar DEFAULT ('BOTH'),
  "isAllowedToOrder" boolean DEFAULT (true),
  "location_types" text[] DEFAULT '{}',
  "location_desc" text[] DEFAULT '{}',
  "created_at" timestamp,
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

CREATE TABLE "presence" (
  "id" varchar PRIMARY KEY,
  "event_id" varchar NOT NULL,
  "user_id" varchar,
  "user_ticket_id" varchar,
  "admin_id" varchar,
  "is_new" boolean DEFAULT (true),
  "created_at" timestamp,
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

CREATE TABLE "comments" (
  "id" varchar PRIMARY KEY,
  "event_id" varchar NOT NULL,
  "user_id" varchar NOT NULL,
  "comment" varchar NOT NULL,
  "like" integer DEFAULT 0,
  "created_at" timestamp,
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

CREATE TABLE "feedback" (
  "id" varchar PRIMARY KEY,
  "event_id" varchar NOT NULL,
  "user_id" varchar NOT NULL,
  "message" varchar NOT NULL,
  "created_at" timestamp,
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

CREATE TABLE "likes" (
  "id" varchar PRIMARY KEY,
  "comment_id" varchar NOT NULL,
  "event_id" varchar NOT NULL,
  "user_id" varchar NOT NULL,
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

CREATE TABLE "rangers" (
  "id" varchar PRIMARY KEY,
  "user_id" varchar NOT NULL,
  "divisi_id" varchar NOT NULL,
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

CREATE TABLE "agenda" (
  "id" varchar PRIMARY KEY,
  "name" varchar NOT NULL,
  "type" varchar NOT NULL,
  "location" varchar NOT NULL,
  "start_at" timestamp NOT NULL,
  "divisi_id" varchar NOT NULL,
  "user_id" varchar NOT NULL,
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

CREATE TABLE "ranger_presences" (
  "id" varchar PRIMARY KEY,
  "ranger_id" varchar NOT NULL,
  "agenda_id" varchar NOT NULL,
  "divisi_id" varchar NOT NULL,
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

CREATE TABLE "tickets" (
  "id" varchar PRIMARY KEY,
  "visibility" varchar DEFAULT ('public'),
  "name" varchar NOT NULL,
  "description" text NOT NULL,
  "price" integer NOT NULL,
  "event_id" varchar NOT NULL,
  "start_at" timestamp NOT NULL,
  "end_at" timestamp NOT NULL,
  "pax_multiplier" integer DEFAULT (1),
  "min_order_pax" integer,
  "max_order_pax" integer,
  "max_pax" int NOT NULL,
  "gender_allowed" varchar DEFAULT ('both'),
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

CREATE TABLE "user_tickets" (
  "id" varchar PRIMARY KEY,
  "public_id" varchar NOT NULL,
  "user_name" varchar NOT NULL,
  "user_email" varchar NOT NULL,
  "user_gender" varchar NOT NULL,
  "user_id" varchar NOT NULL,
  "order_id" varchar NOT NULL,
  "ticket_id" varchar NOT NULL,
  "event_id" varchar NOT NULL,
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

CREATE TABLE "orders" (
  "id" varchar PRIMARY KEY,
  "public_id" varchar NOT NULL,
  "user_id" varchar NOT NULL,
  "payment_method_id" varchar,
  "event_id" varchar,
  "amount" int NOT NULL,
  "donation" integer DEFAULT (0),
  "admin_fee" integer DEFAULT (0),
  "status" varchar DEFAULT ('pending'),
  "invoice_url" text,
  "invoice_image_url" text,
  "expired_at" timestamp,
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

CREATE TABLE "payment_methods" (
  "id" varchar PRIMARY KEY,
  "name" varchar NOT NULL,
  "type" varchar NOT NULL,
  "code" varchar NOT NULL,
  "image_url" text NOT NULL,
  "account_name" varchar,
  "account_number" varchar,
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

ALTER TABLE "presence" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");
ALTER TABLE "presence" ADD FOREIGN KEY ("event_id") REFERENCES "events" ("id");

ALTER TABLE "events" ADD FOREIGN KEY ("divisi_id") REFERENCES "divisi" ("id");

ALTER TABLE "comments" ADD FOREIGN KEY ("event_id") REFERENCES "events" ("id");
ALTER TABLE "comments" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "likes" ADD FOREIGN KEY ("comment_id") REFERENCES "comments" ("id");
ALTER TABLE "likes" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "rangers" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");
ALTER TABLE "rangers" ADD FOREIGN KEY ("divisi_id") REFERENCES "divisi" ("id");

ALTER TABLE "agenda" ADD FOREIGN KEY ("divisi_id") REFERENCES "divisi" ("id");
ALTER TABLE "agenda" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "ranger_presences" ADD FOREIGN KEY ("ranger_id") REFERENCES "rangers" ("id");
ALTER TABLE "ranger_presences" ADD FOREIGN KEY ("agenda_id") REFERENCES "agenda" ("id");
ALTER TABLE "ranger_presences" ADD FOREIGN KEY ("divisi_id") REFERENCES "divisi" ("id");

ALTER TABLE "tickets" ADD FOREIGN KEY ("event_id") REFERENCES "events" ("id");

ALTER TABLE "user_tickets" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");
ALTER TABLE "user_tickets" ADD FOREIGN KEY ("order_id") REFERENCES "orders" ("id");
ALTER TABLE "user_tickets" ADD FOREIGN KEY ("ticket_id") REFERENCES "tickets" ("id");

ALTER TABLE "orders" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");
ALTER TABLE "orders" ADD FOREIGN KEY ("payment_method_id") REFERENCES "payment_methods" ("id");