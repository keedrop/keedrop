// Enables testing the "noscript" variant of the page.
// See https://github.com/cypress-io/cypress/issues/1611

// Unfortunately the old workaround is broken with current versions of
// Cypress / Chromium 🙈

// Cypress.Commands.overwrite('visit', (orig, url, options = {}) => {
//   const parentDocument = cy.state("window").parent.document
//   const iframe = parentDocument.querySelector(".iframes-container iframe")
//   if (false === options.script) {
//     if (false !== Cypress.config("chromeWebSecurity")) {
//       throw new TypeError("When you disable script you also have to set 'chromeWebSecurity' in your config to 'false'")
//     }
//     iframe.sandbox = ""
//   } else {
//     // In case it was added by a visit before, the attribute has to be removed from the iframe
//     iframe.removeAttribute("sandbox")
//   }
//   return orig(url, options);
// })

// Cypress.Commands.add("visitAsHtml", (route) => {
//   cy.request(route)
//     .its("body")
//     .then((html) => {
//       // remove the application code JS bundle
//       html = html.replace(
//         /<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi,
//         ""
//       );
//       cy.document().invoke({ log: false }, "open");
//       cy.document().invoke({ log: false }, "write", html);
//       cy.document().invoke({ log: false }, "close");
//     });
// });
