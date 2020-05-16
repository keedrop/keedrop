// ***********************************************
// This example commands.js shows you how to
// create various custom commands and overwrite
// existing commands.
//
// For more comprehensive examples of custom
// commands please read more here:
// https://on.cypress.io/custom-commands
// ***********************************************
//
//
// -- This is a parent command --
// Cypress.Commands.add("login", (email, password) => { ... })
//
//
// -- This is a child command --
// Cypress.Commands.add("drag", { prevSubject: 'element'}, (subject, options) => { ... })
//
//
// -- This is a dual command --
// Cypress.Commands.add("dismiss", { prevSubject: 'optional'}, (subject, options) => { ... })
//
//
// -- This will overwrite an existing command --
// Cypress.Commands.overwrite("visit", (originalFn, url, options) => { ... })
Cypress.Commands.overwrite('visit', (orig, url, options = {}) => {
  const parentDocument = cy.state("window").parent.document
  const iframe = parentDocument.querySelector(".iframes-container iframe")
  if (false === options.script) {
    if (false !== Cypress.config("chromeWebSecurity")) {
      throw new TypeError("When you disable script you also have to set 'chromeWebSecurity' in your config to 'false'")
    }
    iframe.sandbox = ""
  } else {
    // In case it was added by a visit before, the attribute has to be removed from the iframe
    iframe.removeAttribute("sandbox")
  }
  return orig(url, options);
})
