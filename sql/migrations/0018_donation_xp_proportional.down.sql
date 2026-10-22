-- Kembalikan konfigurasi bawaan sebelum reward proporsional.
UPDATE xp_rules
SET xp = 50,
    description = 'Donasi terkonfirmasi (maksimum 3 per bulan)',
    monthly_cap = 3
WHERE source_type = 'donation';
