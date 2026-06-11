package services

import (
	"html"
	"strconv"
	"strings"
	"time"
)

// brandedEmailHTML enveloppe un contenu transactionnel dans la coquille de
// marque Breezy (clear mode). Layout table + styles 100 % inline pour rester
// lisible partout (Outlook desktop inclus), dégradé de marque violet → indigo →
// cyan (#8D3DFF → #5B6CFF → #47D9FF) et bouton « bulletproof » (repli couleur
// solide #5B6CFF là où le dégradé n'est pas supporté). Fonction pure et testable.
//
//   - heading  : titre principal (gras)
//   - intro    : phrase d'accroche — sert AUSSI de pré-en-tête masqué (aperçu
//     dans la liste des mails) ;
//   - ctaLabel : libellé du bouton d'action ;
//   - ctaURL   : URL du bouton (token déjà encodé), réaffichée en repli copiable ;
//   - footnote : mention sécurité / expiration en pied de carte.
func brandedEmailHTML(baseURL, heading, intro, ctaLabel, ctaURL, footnote string) string {
	logo := strings.TrimRight(baseURL, "/") + "/logo_breezy.png"
	return strings.NewReplacer(
		"{{LOGO}}", html.EscapeString(logo),
		"{{HEADING}}", html.EscapeString(heading),
		"{{INTRO}}", html.EscapeString(intro),
		"{{CTA_LABEL}}", html.EscapeString(ctaLabel),
		"{{CTA_URL}}", ctaURL,
		"{{FOOTNOTE}}", html.EscapeString(footnote),
		"{{YEAR}}", strconv.Itoa(time.Now().Year()),
	).Replace(emailShell)
}

// emailShell : gabarit HTML de la coquille Breezy. Les jetons {{...}} sont
// substitués par brandedEmailHTML. Pas de fmt.Sprintf ici (le CSS contient des
// `%` et `{}` qui casseraient le formatage) : substitution littérale.
const emailShell = `<!DOCTYPE html>
<html lang="fr">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<title>Breezy</title>
</head>
<body style="margin:0; padding:0; width:100%; background-color:#eef1fb; -webkit-text-size-adjust:100%; font-family:'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
  <div style="display:none; max-height:0; overflow:hidden; opacity:0; mso-hide:all;">{{INTRO}}</div>
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#eef1fb;">
    <tr>
      <td align="center" style="padding:32px 16px;">
        <table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px; max-width:600px; background-color:#ffffff; border-radius:24px; overflow:hidden; box-shadow:0 18px 50px rgba(91,108,255,0.18);">
          <tr>
            <td align="center" bgcolor="#5B6CFF" style="background-color:#5B6CFF; background-image:linear-gradient(120deg,#8D3DFF 0%,#5B6CFF 52%,#47D9FF 100%); padding:34px 24px;">
              <img src="{{LOGO}}" alt="Breezy" height="38" style="height:38px; width:auto; display:block; border:0; outline:none; text-decoration:none;">
            </td>
          </tr>
          <tr>
            <td style="padding:42px 44px 6px 44px;">
              <h1 style="margin:0; font-size:26px; line-height:1.25; color:#161a3a; font-weight:700;">{{HEADING}}</h1>
              <p style="margin:16px 0 0 0; font-size:16px; line-height:1.62; color:#4a4f6b;">{{INTRO}}</p>
            </td>
          </tr>
          <tr>
            <td align="center" style="padding:28px 44px 6px 44px;">
              <table role="presentation" cellpadding="0" cellspacing="0" border="0">
                <tr>
                  <td align="center" bgcolor="#5B6CFF" style="border-radius:14px; background-color:#5B6CFF; background-image:linear-gradient(90deg,#8D3DFF 0%,#5B6CFF 55%,#47D9FF 100%);">
                    <a href="{{CTA_URL}}" target="_blank" style="display:inline-block; padding:15px 36px; font-size:16px; font-weight:700; color:#ffffff; text-decoration:none; border-radius:14px;">{{CTA_LABEL}}&nbsp;&nbsp;&rarr;</a>
                  </td>
                </tr>
              </table>
            </td>
          </tr>
          <tr>
            <td style="padding:20px 44px 0 44px;">
              <p style="margin:0; font-size:13px; line-height:1.5; color:#8a8fab;">Le bouton ne fonctionne pas&nbsp;? Copie ce lien dans ton navigateur&nbsp;:</p>
              <p style="margin:6px 0 0 0; font-size:13px; line-height:1.5; word-break:break-all;"><a href="{{CTA_URL}}" target="_blank" style="color:#5B6CFF; text-decoration:underline;">{{CTA_URL}}</a></p>
            </td>
          </tr>
          <tr>
            <td style="padding:28px 44px 0 44px;">
              <div style="height:1px; background-color:#ecedfa; line-height:1px; font-size:0;">&nbsp;</div>
            </td>
          </tr>
          <tr>
            <td style="padding:18px 44px 38px 44px;">
              <p style="margin:0; font-size:13px; line-height:1.6; color:#9a9fbc;">{{FOOTNOTE}}</p>
            </td>
          </tr>
        </table>
        <table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px; max-width:600px;">
          <tr>
            <td align="center" style="padding:22px 24px;">
              <p style="margin:0; font-size:12px; line-height:1.5; color:#a7abc7;">Breezy &middot; {{YEAR}} &mdash; le r&eacute;seau social qui te ressemble.</p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`
