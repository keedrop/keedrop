context("Language relations", function () {
  it("should show language bar", function() {
    cy.visit("/");
    cy.contains("en de ru");
  });

  it("should contain the hash in the links", function() {
    const hash = "#o2WyPtyoQp_WZo8ksib6SaVx70k!GnehvNFAveWkW3uI4IXnjipgyk=";
    cy.visit("/r" + hash);
    cy.get(".languages a").should("have.attr", "href").and("contains", "/r/" + hash);
  });
});
