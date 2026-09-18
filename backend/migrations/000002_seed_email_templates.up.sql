-- Fas 3: standardmallar (engelska) för de fem transaktionsmailen.
-- Platshållare renderas som enkel strängersättning av {key}, inte som körbara
-- mallar — se internal/service/email_service.go.
-- E'' strings are used so \n is interpreted as a real newline.

INSERT INTO email_templates (type, lang, subject, body) VALUES
('sender', 'en', 'Your files are ready to share',
 E'Hi,\n\nYour upload is ready. Share this link with anyone you want to give access to your files:\n\n{download_url}\n\nFiles: {file_names}\nTotal size: {size}\n\nManage or delete this upload any time using the same link.\n\nThanks for using {site_name}.'),

('receiver', 'en', '{email_from} shared files with you',
 E'Hi,\n\n{email_from} shared the following files with you via {site_name}:\n\n{file_names} ({size})\n\n{message}\n\nDownload them here:\n{download_url}'),

('downloaded', 'en', 'Your files were downloaded',
 E'Hi,\n\nYour shared files ({file_names}) were just downloaded{by_email}.\n\nThanks for using {site_name}.'),

('destroyed', 'en', 'Your upload has been deleted',
 E'Hi,\n\nYour upload ({file_names}) has been deleted, either because it expired, was downloaded, or was manually removed.\n\nThanks for using {site_name}.'),

('email_verify', 'en', 'Your verification code',
 E'Hi,\n\nYour verification code is: {code}\n\nEnter this code to continue your upload on {site_name}. The code expires in 1 hour.')
ON CONFLICT (type, lang) DO NOTHING;
