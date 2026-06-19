import paramiko
import time
import os

host = '192.168.1.199'
user = 'abhi-workstation'
password = '1010'

ssh = paramiko.SSHClient()
ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
print("Connecting...")
ssh.connect(host, username=user, password=password)
print("Connected!")

# Let's find where the project is
stdin, stdout, stderr = ssh.exec_command('find ~ -name "open_shine" -type d -not -path "*/go/pkg/*" 2>/dev/null')
paths = stdout.read().decode().strip().split('\n')
if not paths or not paths[0]:
    print("Could not find open_shine directory")
else:
    project_dir = paths[0]
    print(f"Project found at: {project_dir}")
    
    # Run git pull
    print("Running git pull...")
    stdin, stdout, stderr = ssh.exec_command(f'cd {project_dir} && git remote prune origin && git gc && git pull')
    print("Git output:", stdout.read().decode())
    print("Git err:", stderr.read().decode())

    # Upload frontend dist
    print("Uploading frontend dist...")
    sftp = ssh.open_sftp()
    
    local_dist = r"C:\Users\theei\open_shine\web-front\dist"
    remote_dist = f"{project_dir}/web-front/dist"
    
    # ensure remote dist exists
    try:
        sftp.mkdir(remote_dist)
    except IOError:
        pass
        
    for root, dirs, files in os.walk(local_dist):
        for fname in files:
            local_path = os.path.join(root, fname)
            # convert local path to remote path
            rel_path = os.path.relpath(local_path, local_dist).replace("\\", "/")
            remote_path = f"{remote_dist}/{rel_path}"
            
            # create subdirs if needed
            remote_dir = os.path.dirname(remote_path)
            try:
                sftp.stat(remote_dir)
            except IOError:
                try:
                    sftp.mkdir(remote_dir)
                except:
                    pass
            
            sftp.put(local_path, remote_path)
    
    sftp.close()
    print("Uploaded frontend dist!")

    # Restart the service by killing the go main process
    print("Restarting service...")
    stdin, stdout, stderr = ssh.exec_command(f'echo "{password}" | sudo -S systemctl restart open_shine || echo "{password}" | sudo -S pkill -f "go run main.go" || pkill -f open_shine_server')
    print("Restart output:", stdout.read().decode())
    print("Restart err:", stderr.read().decode())

ssh.close()
