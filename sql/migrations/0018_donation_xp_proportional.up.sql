-- Reward donasi dihitung per Rp10.000, bukan lagi satu nilai tetap.
-- XP menjadi pengali per unit Rp10.000 dan reward bawaan tidak dibatasi
-- jumlah donasi per bulan.
UPDATE xp_rules
SET xp = 1,
    description = '1 XP per Rp10.000 donasi terkonfirmasi',
    monthly_cap = NULL
WHERE source_type = 'donation';
