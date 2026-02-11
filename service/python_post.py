import requests
import json
import re

def parse_hosts_file(filename):
    """
    return: [{"hostname": "example.com", "ip": "1.2.3.4"}, ...]
    """
    hosts_list = []
    
    try:
        with open(filename, 'r', encoding='utf-8') as file:
            for line_num, line in enumerate(file, 1):

                line = line.strip()
                if not line or line.startswith('#'):
                    continue
                
                match = re.match(r'(\S+)\s+(\S+)', line)
                if match:
                    ip = match.group(1)
                    hostname = match.group(2)
                    hosts_list.append({"hostname": hostname, "ip": ip})
                else:
                    print(f"warning: {line_num} line format error: {line}")
                    
    except FileNotFoundError:
        print(f"ERROR: file {filename} not exists")
        return []
    except Exception as e:
        print(f"Error: file read error {e}")
        return []
    
    return hosts_list

def send_post_request(hosts_data, api_url="http://localhost:8080/api/hosts/"):
    """
    post to api
    """
    payload = {
        "hosts": hosts_data
    }
    
    headers = {
        'Content-Type': 'application/json'
    }
    
    try:
        response = requests.post(api_url, json=payload, headers=headers)
        
        print(f"http_status_code: {response.status_code}")
        print(f"http_response: {response.text}")
        
        if response.status_code // 100 == 2:
            try:
                return response.json()
            except json.JSONDecodeError:
                return {"success": True, "message": "not json format response"}
        else:
            print(f"false status_code: {response.status_code}")
            return None
            
    except requests.exceptions.ConnectionError:
        print("ERROR: Connection server error")
        return None
    except requests.exceptions.Timeout:
        print("ERROR: timeout")
        return None
    except requests.exceptions.RequestException as e:
        print(f"ERROR: internal error {e}")
        return None

def main():
    debug = False
    hosts_filename = "hosts.dnsmasq.conf.3"  # hosts name
    api_url = "http://localhost:8080/api/hosts/"  # API URL
    
    hosts_data = parse_hosts_file(hosts_filename)
    
    if not hosts_data:
        print("file parse error")
        return
    
    print(f"success: total {len(hosts_data)} record")
    
    for i, host in enumerate(hosts_data, 1):
        if debug is True:
            print(f"{i}. IP: {host['ip']} -> hostname: {host['hostname']}")
    
    result = send_post_request(hosts_data, api_url)
    
    if result:
        print("complate!")
        print("server response:", json.dumps(result, indent=2, ensure_ascii=False))
    else:
        print("server connect false!")

if __name__ == "__main__":
    main()
