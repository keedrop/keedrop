context("Heading hierarchy", function () {
  ["/", "/r/", "/imprint", "/de/", "/ru/"].forEach(function(path) {
    it("Has a single h1 with the site name on " + path, function() {
      cy.visit(path);
      cy.get("h1").should("have.length", 1).and("be.visible").invoke("text").then(function(text) {
        expect(text.trim().toLowerCase()).to.eq("keedrop");
      });
    });

    it("Puts the page header right below the h1 on " + path, function() {
      cy.visit(path);
      cy.get("header h1 ~ h2").should("have.length", 1).and("be.visible");
    });
  });
});
