package newsletter

import (
	"fmt"
)

// WrapHTML takes a raw newsletter body and wraps it in a professional email template.
func WrapHTML(bodyHTML string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background-color: #f4f4f5; margin: 0; padding: 0; }
  .container { max-width: 600px; margin: 40px auto; background: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 4px 6px rgba(0,0,0,0.05); }
  .header { background: linear-gradient(135deg, #3b82f6 0%%, #8b5cf6 100%%); padding: 32px 24px; text-align: center; color: white; }
  .header h1 { margin: 0; font-size: 24px; font-weight: 600; letter-spacing: -0.5px; }
  .content { padding: 32px 24px; color: #3f3f46; line-height: 1.6; font-size: 16px; }
  .content a { color: #3b82f6; text-decoration: none; }
  .content img { max-width: 100%%; height: auto; border-radius: 6px; }
  .footer { background: #f4f4f5; padding: 24px; text-align: center; color: #a1a1aa; font-size: 12px; border-top: 1px solid #e4e4e7; }
</style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>Open Shine Newsletter</h1>
    </div>
    <div class="content">
      %s
    </div>
    <div class="footer">
      <p>Sent via Open Shine</p>
      <p>You received this email because you are subscribed to our newsletter.</p>
    </div>
  </div>
</body>
</html>`, bodyHTML)
}
