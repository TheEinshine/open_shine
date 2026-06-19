import os
import time
import subprocess

WATCH_DIRS = ["web-front/src", "web", "newsletter", "db", "mailer"]
WATCH_EXTS = [".go", ".tsx", ".ts", ".css", ".html"]
POLL_INTERVAL = 3  # seconds

def get_latest_mtime():
    latest = 0
    for watch_dir in WATCH_DIRS:
        if not os.path.exists(watch_dir):
            continue
        for root, _, files in os.walk(watch_dir):
            for file in files:
                if any(file.endswith(ext) for ext in WATCH_EXTS):
                    mtime = os.path.getmtime(os.path.join(root, file))
                    if mtime > latest:
                        latest = mtime
                        
    # Also check main.go in root
    if os.path.exists("main.go"):
        m = os.path.getmtime("main.go")
        if m > latest:
            latest = m
            
    return latest

def main():
    print("Auto-deploy watcher started. Monitoring for changes...")
    last_mtime = get_latest_mtime()
    
    while True:
        time.sleep(POLL_INTERVAL)
        current_mtime = get_latest_mtime()
        
        if current_mtime > last_mtime:
            print("\n" + "="*50)
            print("Changes detected! Triggering auto-deploy...")
            last_mtime = current_mtime
            
            # Step 1: Build the frontend
            print("Building frontend...")
            build_res = subprocess.run("npm run build", cwd="web-front", shell=True)
            if build_res.returncode != 0:
                print("Frontend build failed. Waiting for next change...")
                continue
                
            # Step 2: Push changes to git (optional but good practice to sync backend)
            print("Committing to git...")
            subprocess.run("git add . && git commit -m \"Auto-deploy sync\" && git push", shell=True)
            
            # Step 3: Run the deploy script to upload to the server
            print("Deploying to remote server...")
            subprocess.run("python deploy.py", shell=True)
            
            print("Auto-deploy complete! Watching for new changes...")
            print("="*50 + "\n")

if __name__ == "__main__":
    main()
