import requests
import argparse

def getFilesInDirectory(url: str, auth_data: requests.auth.HTTPBasicAuth, dir_path: str = "", quiet: bool = False):
  """Get list of all files in directory recursively from Nexus repository"""
  files = []
  
  # Извлекаем nexus_addr и repo_name из url
  # url format: http://nexus.example.com/repository/myrepo
  parts = url.split('/repository/')
  nexus_addr = parts[0]
  repo_name = parts[1] if len(parts) > 1 else ""
  
  # Конструируем URL для Nexus browse API
  if dir_path:
    browse_url = "{}/service/rest/v1/search/assets?repository={}&name={}/*".format(nexus_addr, repo_name, dir_path)
  else:
    browse_url = "{}/service/rest/v1/search/assets?repository={}".format(nexus_addr, repo_name)
  
  try:
    if not quiet: print("Getting file list from directory '{}'...".format(dir_path or "root"))
    response = requests.get(browse_url, auth=auth_data)
    
    if response.status_code == 200:
      data = response.json()
      for item in data.get('items', []):
        file_path = item.get('path', '')
        if file_path.startswith(dir_path) if dir_path else True:
          files.append(file_path)
      if not quiet: print("Found {} files in directory".format(len(files)))
    else:
      if not quiet: print("Failed to get directory listing: HTTP {}".format(response.status_code))
      
  except requests.exceptions.RequestException as err:
    if not quiet: print("Error getting directory listing: {}".format(err))
    
  return files

def deleteDirFromNexus(url: str, auth_data: requests.auth.HTTPBasicAuth, path: str, dry_run: bool = False, quiet: bool = False):
  """Delete a specific directory from Nexus repository by deleting all files in it"""
  # Убираем слэш в конце пути
  if path.endswith('/') or path.endswith('\\'):
    path = path[:-1]
  
  if not quiet: print("Deleting directory '{}' from repository...".format(path))
  
  # Получаем все файлы в директории
  files = getFilesInDirectory(url, auth_data, path, quiet)
  
  if not files:
    if not quiet: print("No files found in directory '{}'".format(path))
    return
  
  # Удаляем каждый файл
  deleted_count = 0
  for file_path in files:
    if not quiet: print("Deleting file '{}'...".format(file_path))
    deleteFileFromNexus(url=url, auth_data=auth_data, path=file_path, dry_run=dry_run, quiet=quiet)
    deleted_count += 1
  
  if not quiet: print("Directory '{}' deletion completed. {} files processed.".format(path, deleted_count))
  

def deleteFileFromNexus(url: str, auth_data: requests.auth.HTTPBasicAuth, path: str, dry_run: bool = False, quiet: bool = False):
  """Delete a specific file from Nexus repository"""
  file_url = "{}/{}".format(url, path)
  if not dry_run:
    if not quiet: print("File '{}' will be deleted from {}...".format(path, file_url))
    try:
      response = requests.delete(file_url, auth=auth_data)    
      if response.status_code == 404:
        if not quiet: print("File '{}' not found in repository (404)".format(path))
      elif response.status_code == 204:
        if not quiet: print("File '{}' deleted successfully. Elapsed: {}".format(path, response.elapsed))
      else:
        if not quiet: print("Unexpected response code {} for file '{}'".format(response.status_code, path))
    except requests.exceptions.RequestException as err:
      print("Error while deleting file {}: {}".format(path, err))
      exit(1)
  else:
    if not quiet: print("File '{}' planned for deletion from {}.".format(path, file_url))

parser = argparse.ArgumentParser(description='Delete file or directory from Nexus OSS Raw Repository')
parser.add_argument('paths', metavar='path', nargs='+',
                    help='Path of file or directory to delete from repository')
parser.add_argument('-u','--user', 
                    help='User authentification login')
parser.add_argument('-p','--password', 
                    help='User authentification password')
parser.add_argument('-a','--address', 
                    help='Nexus OSS host address. Example: http://nexus.redkit-lab.ru or http://172.23.10.22')
parser.add_argument('-r','--repository', 
                    help='Nexus OSS raw repository name')
parser.add_argument('--dry',
                    default=False,
                    action='store_true',
                    help="Dry run. Files just will be printed, without real deletion from repository.")
parser.add_argument('-q','--quiet',
                    default=False,
                    action='store_true',
                    help="Print only destination URL with 0 exit code, if files deleted successful. Or only error text with non-zero exit code.")


args = vars(parser.parse_args())

nexus_addr = args["address"]
if (nexus_addr[-1] == "/" or nexus_addr[-1] == "\\"):
  nexus_addr = nexus_addr[:-1]

repo_name = args["repository"]
url = "{}/repository/{}".format(nexus_addr, repo_name)
link_url = "{}/#browse/browse:{}".format(nexus_addr, repo_name)

user_login = args["user"]
user_pass = args["password"]
auth_data = requests.auth.HTTPBasicAuth(user_login, user_pass)

dry_run = args["dry"]
quiet = args["quiet"]

for path in args["paths"]:
      if not quiet: print("Process path '{}'".format(path))

      if (path[-1] == "/" or path[-1] == "\\"):
        deleteDirFromNexus(url=url, auth_data=auth_data, path=path, dry_run=dry_run, quiet=quiet)
      else:
        deleteFileFromNexus(url=url, auth_data=auth_data, path=path, dry_run=dry_run, quiet=quiet)

print(link_url)
if not quiet: print("Success!")
exit(0)