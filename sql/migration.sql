CREATE TABLE "users" (
  "id" varchar PRIMARY KEY,
  "name" varchar NOT NULL,
  "username" varchar DEFAULT 'anonim',
  "gender" varchar NOT NULL DEFAULT 'male',
  "age" integer NOT NULL,
  "phone" varchar,
  "email" varchar,
  "password" varchar,
  "address" varchar,
  "role" varchar DEFAULT 'jamaah',
  "created_at" timestamp,
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
  "created_at" timestamp,
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

CREATE TABLE "presence" (
  "id" varchar PRIMARY KEY,
  "event_id" varchar NOT NULL,
  "user_id" varchar NOT NULL,
  "is_new" bolean DEFAULT (true),
  "created_at" timestamp,
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

ALTER TABLE "presence" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "presence" ADD FOREIGN KEY ("event_id") REFERENCES "events" ("id");

ALTER TABLE "events" ADD FOREIGN KEY ("divisi_id") REFERENCES "divisi" ("id");
