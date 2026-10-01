# Included from default.conf
add_header X-Content-Type-Options "nosniff" always;
add_header X-Frame-Options "SAMEORIGIN" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
add_header Permissions-Policy "camera=(), microphone=(), geolocation=()" always;
# The Content-Security-Policy is set as <meta> tag by Hugo (layouts/partials/head.html),
# so the dev server can allow its live-reload script.
