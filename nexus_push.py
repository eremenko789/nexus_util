import requests
import argparse
import os

def sendFile(url: str, auth_data: requests.auth.HTTPBasicAuth, file_path: str, dest_path: str = "", dry_run: bool = False, quiet: bool = False):
  if len(dest_path) == 0:
    dest_path = file_path
  file_url = "{}/{}".format(url, dest_path)
  if not dry_run:
    if not quiet: print("File '{}' will be pushed as {}...".format(file_path, file_url))
    try:
      response = requests.put(file_url, data=open(file_path,'rb').read(), auth=auth_data)    
      if not quiet: print("Sending file '{}' completed. Elapsed: {}".format(file_path, response.elapsed))
    except requests.exceptions.RequestException as err:
      print("Error while sending file {}: {}".format(file_path, err))
      exit(1)
  else:
    if not quiet: print("File '{}' planned for pushing on {}.".format(file_path, file_url))

def sendDirRecursive(url: str, auth_data: requests.auth.HTTPBasicAuth, path: str, dry_run: bool = False, quiet: bool = False,  relative: bool = False):
  if not quiet: print("Process directory '{}'".format(path))
  for root, directories, files in os.walk(path):
      for file in files:
        file_path = os.path.join(root, file)
        rel_path = os.path.relpath(file_path, path)
        if relative:
          sendFile(url=url, auth_data=auth_data, file_path=file_path, dest_path=rel_path, dry_run=dry_run, quiet=quiet)
        else:
          sendFile(url=url, auth_data=auth_data, file_path=file_path, dry_run=dry_run, quiet=quiet)



parser = argparse.ArgumentParser(description='Publish file or directory to Nexus OSS Raw Repository')
parser.add_argument('paths', metavar='path', nargs='+',
                    help='Path of file or directory to publish')
parser.add_argument('-u','--user', 
                    help='User authentification login')
parser.add_argument('-p','--password', 
                    help='User authentification password')
parser.add_argument('-a','--address', 
                    help='Nexus OSS host address. Example: http://nexus.redkit-lab.ru ot http://172.23.10.22')
parser.add_argument('-r','--repository', 
                    help='Nexus OSS raw repository name')
parser.add_argument('-d','--destination', 
                    help='Destination path into Nexus raw repository')
parser.add_argument('--dry',
                    default=False,
                    action='store_true',
                    help="Dry run. Files just will be printed, without real pushing to repository.")
parser.add_argument('-q','--quiet',
                    default=False,
                    action='store_true',
                    help="Print only destination URL with 0 exit code, if files pushed successful. Or only error text with non-zero exit code.")
parser.add_argument('--relative',
                    default=False,
                    action='store_true',
                    help="""If true, the directory data will be send relative by directory.
                    For example, destination in repo is '/testDir' and we try push directory 'localDir/localSubDir/'. 
                    With --dir-relative flag we'll find file in repo '/testDir/file1.data'. 
                    Without --dir-relative flag we'll find file '/testDir/localDir/localSubDir/file1.data'."""
                    )

args = vars(parser.parse_args())

nexus_addr = args["address"]
if (nexus_addr[-1] == "/" or nexus_addr[-1] == "\\"):
  nexus_addr = nexus_addr[:-1]

repo_name = args["repository"]
destination = args["destination"]
if (destination[-1] == "/" or destination[-1] == "\\"):
  destination = destination[:-1]

url = "{}/repository/{}/{}".format(nexus_addr, repo_name, destination)
link_dest = destination
link_dest.replace("/", "%2F")
link_url = "{}/#browse/browse:{}:{}".format(nexus_addr, repo_name, link_dest)

user_login = args["user"]
user_pass = args["password"]
auth_data = requests.auth.HTTPBasicAuth(user_login, user_pass)

dry_run = args["dry"]
quiet = args["quiet"]
relative = args["relative"]



for path in args["paths"]:
      if not quiet: print("Process path '{}'".format(path))
      is_dir = False

      try:
        if not os.path.exists(path):
            raise argparse.ArgumentTypeError("Path '{}' doesn't exist".format(path))

        if (path[-1] == "/" or path[-1] == "\\"):
          if not os.path.isdir(path):
            raise argparse.ArgumentTypeError("Path '{}' declared as directory is not directory".format(path))
          else:
            is_dir = True
        else:
          if not os.path.isfile(path):
            raise argparse.ArgumentTypeError("Path '{}' is not file.".format(path))
            
      except argparse.ArgumentTypeError as err:
        print("Error while process path {}: {}".format(path, err))
        exit(1)

      if is_dir:
        sendDirRecursive(url=url, auth_data=auth_data, path=path, dry_run=dry_run, quiet=quiet, relative=relative)
      else:
        dest_path = path
        if relative:
          dest_path = os.path.basename(path)
        sendFile(url=url, auth_data=auth_data, file_path=path, dest_path=dest_path, dry_run=dry_run, quiet=quiet)

print(link_url)
exit(0)