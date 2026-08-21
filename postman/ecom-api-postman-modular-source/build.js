#!/usr/bin/env node
/**
 * build.js
 * ---------
 * Menggabungkan collection.manifest.json (file induk) + semua file fitur
 * di collection/<role>/*.json menjadi SATU file Postman collection
 * (dist/Ecom-API.postman_collection.json) yang siap di-import ke Postman.
 *
 * Kenapa perlu ini?
 * Postman tidak punya fitur "collection lintas-file" secara native — satu
 * collection = satu file JSON. Jadi source-nya kita pecah per fitur biar
 * gampang di-maintain, dan file JADI-nya (yang diimport ke Postman) kita
 * generate lewat script ini. Tiap ada perubahan/fitur baru, cukup edit file
 * fitur terkait -> jalankan `node build.js` lagi -> re-import ke Postman.
 *
 * Cara pakai:
 *   node build.js
 *
 * Opsional, custom path:
 *   node build.js path/ke/manifest.json path/output/collection.json
 */

const fs = require("fs");
const path = require("path");

const ROOT = __dirname;
const manifestPath = process.argv[2]
  ? path.resolve(process.argv[2])
  : path.join(ROOT, "collection.manifest.json");
const outputPath = process.argv[3]
  ? path.resolve(process.argv[3])
  : path.join(ROOT, "dist", "Ecom-API.postman_collection.json");

function readJson(filePath) {
  if (!fs.existsSync(filePath)) {
    throw new Error(`File tidak ditemukan: ${filePath}`);
  }
  const raw = fs.readFileSync(filePath, "utf-8");
  try {
    return JSON.parse(raw);
  } catch (err) {
    throw new Error(`Gagal parse JSON di ${filePath}: ${err.message}`);
  }
}

function build() {
  console.log(`\n📖 Membaca manifest: ${path.relative(ROOT, manifestPath)}`);
  const manifest = readJson(manifestPath);

  if (!Array.isArray(manifest.roles) || manifest.roles.length === 0) {
    throw new Error("Manifest tidak punya 'roles' — tidak ada yang bisa digabung.");
  }

  const roleFolders = manifest.roles.map((role) => {
    if (!role.folder) throw new Error("Setiap entry di 'roles' wajib punya 'folder'.");
    if (!Array.isArray(role.features)) {
      throw new Error(`Role '${role.folder}' wajib punya array 'features'.`);
    }

    console.log(`\n📁 Role: ${role.folder}`);
    const featureFolders = role.features.map((relPath) => {
      const abs = path.join(ROOT, relPath);
      console.log(`   └─ 🧩 ${relPath}`);
      const feature = readJson(abs);

      if (!feature.name || !Array.isArray(feature.item)) {
        throw new Error(`File fitur '${relPath}' wajib punya 'name' (string) dan 'item' (array).`);
      }

      const folder = {
        name: feature.name,
        item: feature.item,
      };
      if (feature.description) folder.description = feature.description;
      if (feature.auth) folder.auth = feature.auth;
      return folder;
    });

    const roleFolder = {
      name: role.folder,
      item: featureFolders,
    };
    if (role.description) roleFolder.description = role.description;
    if (role.auth) roleFolder.auth = role.auth;
    return roleFolder;
  });

  const finalCollection = {
    info: manifest.info,
    item: roleFolders,
  };
  if (manifest.auth) finalCollection.auth = manifest.auth;
  if (manifest.event) finalCollection.event = manifest.event;
  if (manifest.variable) finalCollection.variable = manifest.variable;

  fs.mkdirSync(path.dirname(outputPath), { recursive: true });
  fs.writeFileSync(outputPath, JSON.stringify(finalCollection, null, 2) + "\n", "utf-8");

  const totalRequests = roleFolders.reduce(
    (sum, role) => sum + role.item.reduce((s, feat) => s + feat.item.length, 0),
    0
  );

  console.log(`\n✅ Berhasil digabung!`);
  console.log(`   Role folders : ${roleFolders.length}`);
  console.log(`   Total request: ${totalRequests}`);
  console.log(`   Output       : ${path.relative(ROOT, outputPath)}\n`);
}

try {
  build();
} catch (err) {
  console.error(`\n❌ Build gagal: ${err.message}\n`);
  process.exit(1);
}
