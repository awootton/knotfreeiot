
### what

The monitor pod is a deployment in the k8s that runs some clients like get-unix-time


### howto: deploy
``` 
docker build -t  gcr.io/fair-theater-238820/monitor_pod .	
docker push gcr.io/fair-theater-238820/monitor_pod 

kk apply -f deploy.yaml

```

### howto:  test locally 

// knotfreeserver.com is localhost in /etc/hosts Can also use knotfree.net
// just paste the whole thing into a terminal

nc knotfree.net 7465
C token "eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2NTEyODUzMzMsImlzcyI6Il85c2giLCJqdGkiOiJTVnJTNUlBYy03eklyNDgxMU1JeEYydDkiLCJpbiI6MTAwMDAwMCwib3V0IjoxMDAwMDAwLCJzdSI6MjAwMDAwLCJjbyI6MjAwMDAwLCJ1cmwiOiJrbm90ZnJlZS5uZXQifQ.wXgyiPWM7xNnpL_Ihvs3reCsRKWZC0zqIVPrbMPe30h20vHpiBn8jbtw_mcAaHe3mJdCbXgXkY7u_nIgO7C7Cg"
S myaddresstopicchannel
P get-unix-time myaddresstopicchannel "get time"


