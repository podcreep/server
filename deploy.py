
import argparse
import os
import platform
import shutil
import subprocess
import zipfile

parser = argparse.ArgumentParser(description='Run the podcreep server locally.')
parser.add_argument('--web_path', type=str, default='../web', help='Path where the web app is checked out to.')
parser.add_argument('--server_path', type=str, default='.', help='Path where the server code is checked out to.')
parser.add_argument('--android_path', type=str, default='../android', help='Path where the Android app\'s code is checked out to.')
parser.add_argument('--deploy_path', type=str, default='../dist', help='Path where we build and deploy the server to, temporarily.')
parser.add_argument('--keystore_path', type=str, default='../keystore.jks', help='Path to the keystore path.')
parser.add_argument('--keystore_pass', type=str, default='', help='Password for the keystore.')
parser.add_argument('--server_dest', type=str, required=True, help='Location (in \'scp\' format) we copy the server.zip to. e.g. username@host:/path/file.zip.')
args = parser.parse_args()

web_path = os.path.abspath(args.web_path)
server_path = os.path.abspath(args.server_path)
android_path = os.path.abspath(args.android_path)
keystore_path = os.path.abspath(args.keystore_path)
deploy_path = os.path.abspath(args.deploy_path)

# In addition to .go and .py source files, which we ignore when creating the server dist folder,
# we'll also ignore this list of files.
SERVER_IGNORE_FILES = [
  ".gitignore",
  "go.mod",
  "go.sum",
  "LICENSE",
  "README.md",
]

ANDROID_AAB_PATH = os.path.join(android_path, "mobile/build/outputs/bundle/release/mobile-release.aab")

def build_web():
  print(" - building web...")

  # First, clear the old dist/ directory
  dist_dir = os.path.join(web_path, 'dist')
  for f in os.listdir(dist_dir):
    os.remove(os.path.join(dist_dir, f))

  # Now run 'ng build' to actually build the stuff.
  subprocess.run("ng build --configuration production", cwd=web_path, check=True, shell=True)


def copy_web():
  print(" - copying web...")

  # First, clear the old dist/ directory
  dest_dir = os.path.join(server_path, 'dist')
  for f in os.listdir(dest_dir):
    os.remove(os.path.join(dest_dir, f))

  # Now copy the files.
  src_dir = os.path.join(web_path, 'dist')
  for f in os.listdir(src_dir):
    shutil.copy(os.path.join(src_dir, f), dest_dir)


def build_server():
  print(" - building server")

  env = os.environ
  env["GOOS"] = "linux"
  subprocess.run("go build", cwd=server_path, check=True, shell=True, env=env)


def copy_server():
  print(" - copying server")

  server_deploy_path = os.path.join(deploy_path, "server")
  if os.path.isdir(server_deploy_path):
    # If it already exists, delete it first.
    shutil.rmtree(server_deploy_path)

  os.makedirs(server_deploy_path)
  shutil.move(os.path.join(server_path, "server"), os.path.join(server_deploy_path, "podcreep"))
  for root, _, files in os.walk(server_path):
    for f in files:
      rel_dir = os.path.relpath(root, server_path)
      if rel_dir.startswith(".git"):
        # Ignore files in the .git directory
        continue
      _, ext = os.path.splitext(f)
      if ext == ".go" or ext == ".py":
        # Ignore go and python source files.
        continue
      src_file = f
      if rel_dir != ".":
        src_file = os.path.join(rel_dir, f)
      if src_file in SERVER_IGNORE_FILES:
        # Ignore other files in SERVER_IGNORE_FILES.
        continue
      dest_dir = os.path.join(server_deploy_path, rel_dir)
      if not os.path.exists(dest_dir):
        os.makedirs(dest_dir)
      shutil.copy(os.path.join(root, f), dest_dir)


def build_android():
  print(" - building android")
  print("   - clean")
  subprocess.run([os.path.join(".", "gradlew"), "clean"], cwd=android_path, check=True, shell=True)
  print("   - bundle")
  subprocess.run([os.path.join(".", "gradlew"), "bundle"], cwd=android_path, check=True, shell=True)


def sign_android():
  print(" - signing android app")
  subprocess.run([
      "jarsigner", "-verbose", "-sigalg", "SHA256withRSA", "-digestalg", "SHA-256",
      "-keystore", keystore_path, ANDROID_AAB_PATH, "Codeka", "-storepass", args.keystore_pass])


def get_app_version():
  bundletool_path = os.path.join(android_path, "bundletool-all-1.8.2.jar")
  proc = subprocess.run([
      "java", "-jar", bundletool_path, "dump", "manifest", "--bundle", ANDROID_AAB_PATH,
      "--xpath", "/manifest/@android:versionName"
    ], capture_output=True, check=True)
  version = proc.stdout.decode("utf-8")
  return version.strip()


def copy_android():
  version = get_app_version()
  dest_file = f"podcreep-{version}.aab"
  print(" - copying to", dest_file)
  dest_dir = os.path.join(deploy_path, "android")
  if not os.path.exists(dest_dir):
    os.makedirs(dest_dir)
  shutil.copy(ANDROID_AAB_PATH, os.path.join(dest_dir, dest_file))


def zip_server():
  print(" - zipping server")
  with zipfile.ZipFile(os.path.join(deploy_path, "server.zip"), "w") as server_zip:
    server_deploy_path = os.path.join(deploy_path, "server")
    for root, _, files in os.walk(server_deploy_path):
      for f in files:
        full_path = os.path.join(root, f)
        zip_path = os.path.relpath(full_path, server_deploy_path)
        server_zip.write(full_path, zip_path)


def deploy_server():
  print(" - deploying server")
  subprocess.run(["scp", os.path.join(deploy_path, "server.zip"), args.server_dest])


def main():
  build_web()
  copy_web()
  build_server()
  copy_server()
  build_android()
  sign_android()
  copy_android()
  zip_server()
  deploy_server()


if __name__ == "__main__":
  main()