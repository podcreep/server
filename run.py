
import argparse
import os
import signal
import subprocess
import sys
from time import sleep

parser = argparse.ArgumentParser(description='Run the podcreep server locally.')
parser.add_argument('--dbuser', type=str, default='podcreep_user', help='Username for the database')
parser.add_argument('--dbpass', type=str, default='', help='Password to use for the database user.')
parser.add_argument('--dbname', type=str, default='podcreep', help='Name of the database to connect to.')
parser.add_argument('--dbhost', type=str, default='localhost', help='Host of the database server.')
parser.add_argument('--blob_store_path', type=str, default='../store', help='Path to a directory on disk where we\'ll store "blobs", i.e. icons etc.')
parser.add_argument('--admin_password', type=str, default='secret', help='Password to access the admin section.')
args = parser.parse_args()

# If we get a sigint when this is true, we'll exit. Otherwise, ignore the signal.
exit_on_sigint = False
def sigint_handler(signal, frame):
  if exit_on_sigint:
    sys.exit(0)
signal.signal(signal.SIGINT, sigint_handler)

def build_and_run_server():
  subprocess.run(['adb','reverse','tcp:8080','tcp:8080'])
  
  env = os.environ.copy()
  env['DATABASE_URL'] = f'postgres://{args.dbuser}:{args.dbpass}@{args.dbhost}/{args.dbname}'
  env['BLOB_STORE_PATH'] = args.blob_store_path
  env['DEBUG'] = '1'
  env['ADMIN_PASSWORD'] = args.admin_password

  subprocess.run(['go','run','main.go'], check=False, env=env)

while True:
  build_and_run_server()

  print('Process exited, waiting 3 seconds and starting again. Press Ctrl+C to exit.')
  exit_on_sigint = True
  sleep(3)
  exit_on_sigint = False
