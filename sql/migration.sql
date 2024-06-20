CREATE TABLE "users" (
  "id" varchar PRIMARY KEY,
  "name" varchar NOT NULL,
  "username" varchar DEFAULT ('anonim'),
  "gender" varchar NOT NULL DEFAULT ('male'),
  "age" integer NOT NULL,
  "phone" varchar,
  "email" varchar,
  "password" varchar,
  "address" varchar,
  "role" varchar DEFAULT ('jamaah'),
  "activity" varchar DEFAULT ('pelajar'),
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
  "participant" integer DEFAULT 0,
  "created_at" timestamp,
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

CREATE TABLE "presence" (
  "id" varchar PRIMARY KEY,
  "event_id" varchar NOT NULL,
  "user_id" varchar NOT NULL,
  "is_new" bool DEFAULT (true),
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

INSERT INTO public.divisi (id, name, regional, created_at, updated_at, deleted_at) VALUES ('1', 'Sports', 'Solo', '2024-01-03 21:18:47.171529', '2024-01-03 21:18:47.171529', null);
INSERT INTO public.divisi (id, name, regional, created_at, updated_at, deleted_at) VALUES ('2', 'Kajian Pekanan', 'Solo', '2024-01-03 21:18:47.171529', '2024-01-03 21:18:47.171529', null);
INSERT INTO public.divisi (id, name, regional, created_at, updated_at, deleted_at) VALUES ('3', 'KEY', 'Solo', '2024-01-03 21:18:47.171529', '2024-01-03 21:18:47.171529', null);

INSERT INTO public.events (id, slug, code, title, "desc", image_url, speaker, divisi_id, start_at, end_at, participant, created_at, updated_at, deleted_at) VALUES ('67f40c67-db7e-4854-89c1-d24f68e1112e', 'rasan-rasan-4a448', '20240102', 'Air Mata Diujung Sajadah Eps. 2', 'Air Mata Diujung Sajadah Eps. 2', 'https://scontent-cgk1-2.cdninstagram.com/v/t51.2885-15/413893817_1104389514062218_8435652700476609878_n.webp?stp=dst-jpg_e35&efg=eyJ2ZW5jb2RlX3RhZyI6ImltYWdlX3VybGdlbi4xMDgweDEwODAuc2RyIn0&_nc_ht=scontent-cgk1-2.cdninstagram.com&_nc_cat=110&_nc_ohc=QIG6esl36tQAX--Nvd8&edm=ACWDqb8BAAAA&ccb=7-5&ig_cache_key=MzI2NjgxODcyNDA1MDkwNTAwMg%3D%3D.2-ccb7-5&oh=00_AfADSEC7LdC2fraWsBUQz26zmY7eoLhOGZlhM7RE1eJobA&oe=659398C3&_nc_sid=ee9879', 'Ustadz Iqbal Tantowi', '2', '2024-01-01 13:16:10.802807', '2024-01-07 13:16:10.802807', 0, '2024-01-03 21:19:07.202963', '2024-01-06 22:28:42.075822', null);

INSERT INTO public.users ("id", "name", "username", "gender", "age", "phone", "email", "password", "address", "role") VALUES ('a1f2661a-4ea3-4fd8-9058-6992d2be4213', 'PJ KEY', 'PJ KEY', 'male', 24, '08123456789', 'pj_key@ynsolo.id', '$2a$10$qIMxXBYVLUGH4iYfkE0F5efrd9/8/HPssuVeakd7Q5mbb8v.oL.J.', 'Solo', 'pj');
INSERT INTO public.users ("id", "name", "username", "gender", "age", "phone", "email", "password", "address", "role") VALUES ('nbdasdad-t2d3-4fd8-9058-6992d2be4213', 'Ranger KEY', 'Ranger KEY', 'male', 24, '081123456789', 'ranger_key@ynsolo.id', '$2a$10$qIMxXBYVLUGH4iYfkE0F5efrd9/8/HPssuVeakd7Q5mbb8v.oL.J.', 'Solo', 'ranger');

INSERT INTO public.rangers ("id", "user_id", "divisi_id") VALUES ('92479cae-e649-4a2b-b013-4565e6c1c823','nbdasdad-t2d3-4fd8-9058-6992d2be4213', '3')