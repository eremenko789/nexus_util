import requests
import argparse
import os

def download_url_and_save(url: str, dest_path: str, root: str = ""):
  if not QUIET: print("REST API: '{}'".format(url))
  if not QUIET: print("DESTINATION: {}".format(dest_path))
  try:
    os.makedirs(os.path.dirname(dest_path), exist_ok=True)
    with open(dest_path, 'wb') as f:
      r = requests.get(url, allow_redirects=True, auth=AUTH_DATA)
      f.write(r.content)
  except Exception as err: 
    if not QUIET: print("Failure ({})".format(err))
    exit(1)
  else:
    if not QUIET: print("Success file download...")

# 'http://172.23.10.22:8082/service/rest/v1/search/assets/download?repository=redkitdoc&name=scada%2Frelease%2F2005%2Flatest%2Farm-redkit-scada.pdf'
def download_file(file_path: str, dest_path: str = ""):
  if not QUIET: print("Download file {} ...".format(file_path))
  if REMOTE_ROOT is None or file_path.startswith(REMOTE_ROOT):
    name = file_path
  else:
    name = "{}/{}".format(REMOTE_ROOT, file_path)
  name = name.replace("/", "%2F")
  url = "{}/service/rest/v1/search/assets/download?repository={}&name={}".format(NEXUS_ADDR, REPOSITORY, name)
  file_name = file_path.split('/')[-1]
  if len(dest_path) == 0:
    dest_path = "{}/{}".format(DESTINATION, file_name)  
  if not DRY_RUN: download_url_and_save(url=url, dest_path=dest_path)
  if not QUIET: print("Success file {} ...".format(file_path))


def get_all_files(src: str):
  files = []
  try:
    name_query = "{}*".format(src) if REMOTE_ROOT is None else "{}/{}*".format(REMOTE_ROOT, src)
    base_query = "repository={}&name={}".format(REPOSITORY, name_query).replace("/", "%2F")
    has_more_data = True
    continuationToken = ''
    while has_more_data:
      query = base_query
      if len(continuationToken) > 0:
        query += "&continuationToken={}".format(continuationToken)

      # 'http://172.23.10.22:8082/service/rest/v1/search/assets?repository=redkitdoc&name=scada%2Frelease%2F2005%2F*'
      url = "{}/service/rest/v1/search/assets?{}".format(NEXUS_ADDR, query)
      if not QUIET: print("REST API request: '{}'".format(url))
      r = requests.get(url, allow_redirects=True, auth=AUTH_DATA)
      
      if not QUIET: print("REST API response: '{}'".format(r))
      response_json = r.json()
      if response_json['continuationToken'] is None:
        has_more_data = False
        continuationToken = ''
      else:
        has_more_data = True
        continuationToken = response_json['continuationToken']
      
      for item in response_json['items']:
        files.append(item['path'])
      
  except Exception as err:
    if not QUIET: print("Failure getting directory files ({})".format(err))
    exit(1)
  else:
    if not QUIET: print("Directory '{}' contains {} files.".format(src, len(files)))
    return files


  


def download_dir(src: str):
  if not QUIET: print("Download dir {} ...".format(src))
  files = get_all_files(src)
  for file in files:
    print("file '{}' searched".format(file))
    dest_path = "{}/{}".format(DESTINATION, file[len(REMOTE_ROOT) if REMOTE_ROOT is not None else 0 :])
    download_file(file, dest_path)
  if not QUIET: print("Success dir {} ...".format(src))
  

parser = argparse.ArgumentParser(description='Pull files or directory from Nexus OSS Raw Repository')
parser.add_argument('-a','--address', 
                    required=True,
                    help='Nexus OSS host address. Example: http://nexus.redkit-lab.ru or http://172.23.10.22')
parser.add_argument('-r','--repository', 
                    required=True,
                    help='Nexus OSS raw repository name')
parser.add_argument('-u','--user', 
                    help='User authentification login')
parser.add_argument('-p','--password', 
                    help='User authentification password')
parser.add_argument('-d','--destination',
                    required=True,
                    help='Path to save files')
parser.add_argument('--root', 
                    help='Root in Nexus repository')
parser.add_argument('--dry',
                    default=False,
                    action='store_true',
                    help="Dry run. Files just will be printed, without real pushing to repository.")
parser.add_argument('-q','--quiet',
                    default=False,
                    action='store_true',
                    help="Print only destination URL with 0 exit code, if files pushed successful. Or only error text with non-zero exit code.")
parser.add_argument('sources', metavar='source', nargs='+',
                    help='Path of file or directory in Nexus repository')

args = vars(parser.parse_args())
print(args)

DRY_RUN = args["dry"]
QUIET = args["quiet"]
NEXUS_ADDR = args["address"]
if (NEXUS_ADDR[-1] == "/" or NEXUS_ADDR[-1] == "\\"):
  NEXUS_ADDR = NEXUS_ADDR[:-1]
REPOSITORY = args["repository"]
user_login = args["user"]
user_pass = args["password"]
AUTH_DATA = requests.auth.HTTPBasicAuth(user_login, user_pass)
DESTINATION = args["destination"]
if (DESTINATION[-1] == "/" or DESTINATION[-1] == "\\"):
  DESTINATION = DESTINATION[:-1]
REMOTE_ROOT = args["root"]
if REMOTE_ROOT is not None and (REMOTE_ROOT[-1] == "/" or REMOTE_ROOT[-1] == "\\"):
  REMOTE_ROOT = REMOTE_ROOT[:-1]



# Проверим существование директории сохранения
if not os.path.exists(DESTINATION):
  raise argparse.ArgumentTypeError("Destination path '{}' doesn't exist".format(DESTINATION))
if not os.path.isdir(DESTINATION):
  raise argparse.ArgumentTypeError("Destination path '{}' is not directory".format(DESTINATION))

for source in args["sources"]:
  if not QUIET: print("Process source '{}'".format(source))
  is_dir = False

  if (source[-1] == "/" or source[-1] == "\\"):
    is_dir = True

  if is_dir:
    if not QUIET: print("source '{}' is directory".format(source))
    download_dir(src=source)
  else:
    if not QUIET: print("source '{}' is file".format(source))
    download_file(file_path=source)
  
if not QUIET: print("Success!")