
## Tickets aka tokens used for access defined here.

See: `type KnotFreePayload struct`

and `MakeTicket(data *KnotFreePayload, privateKey []byte) ([]byte, error)`

and `func VerifyTicket(ticket []byte, publicKey []byte) (*KnotFreePayload, bool)`


<!-- Global site tag (gtag.js) - Google Analytics -->
<script async src="https://www.googletagmanager.com/gtag/js?id=UA-156005349-2"></script>
<script>
  window.dataLayer = window.dataLayer || [];
  function gtag(){dataLayer.push(arguments);}
  gtag('js', new Date());

  gtag('config', 'UA-156005349-2');
</script>
