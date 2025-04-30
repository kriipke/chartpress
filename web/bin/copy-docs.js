const fs = require('fs');
const path = require('path');

// Define source and destination paths
const SOURCE_DIR = path.join(__dirname, '..', '..', 'docs');
const DEST_DIR = path.join(__dirname, '..', 'public', 'docs');

// Copy files recursively from source to destination
function copyFilesRecursive(src, dest) {
  if (!fs.existsSync(dest)) {
    fs.mkdirSync(dest, { recursive: true });
  }

  const items = fs.readdirSync(src, { withFileTypes: true });

  items.forEach((item) => {
    const srcPath = path.join(src, item.name);
    const destPath = path.join(dest, item.name);

    if (item.isDirectory()) {
      copyFilesRecursive(srcPath, destPath);
    } else {
      fs.copyFileSync(srcPath, destPath);
      console.log(`Copied: ${srcPath} -> ${destPath}`);
    }
  });
}

// Perform the copy
try {
  console.log('Copying documentation files...');
  copyFilesRecursive(SOURCE_DIR, DEST_DIR);
  console.log('Documentation files copied successfully.');
} catch (error) {
  console.error('Error copying documentation files:', error);
  process.exit(1);
}
