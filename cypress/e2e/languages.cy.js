context("Language relations", function () {
  it("should show language bar", function() {
    cy.visit("/");
    cy.get(".languages span").should("contain", "en");
    cy.get(".languages a").should("have.text", "deru");
  });

  it("should contain the hash in the links", function() {
    const hash = "#o2WyPtyoQp_WZo8ksib6SaVx70k!GnehvNFAveWkW3uI4IXnjipgyk=";
    cy.visit("/r" + hash);
    cy.get(".languages a").should("have.attr", "href").and("contains",  hash);
  });

  it("should have correct translated links in language bar on each page", function() {
    cy.visit("/imprint");
    cy.get(".languages a[hreflang=de]").should("have.attr", "href").and("contains", "impressum");
  });

  it("should point search engines to absolute URLs of every translation", function() {
    cy.visit("/imprint");
    cy.get("head link[rel=alternate][hreflang]").should("have.length", 3).each(($link) => {
      expect($link.attr("href")).to.match(/^https?:\/\//);
    });
  });
});
