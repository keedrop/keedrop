const pages = ["/", "/imprint", "/r", "/de/", "/de/impressum", "/de/r", "/ru/", "/ru/imprint", "/ru/r"];

context("Accessibility", function () {
  pages.forEach((page) => {
    it(`should have readable contrast and large enough tap targets on ${page}`, function () {
      cy.visit(page);
      cy.injectAxe();
      cy.checkA11y(null, { runOnly: { type: "rule", values: ["color-contrast", "target-size"] } });
    });
  });
});
