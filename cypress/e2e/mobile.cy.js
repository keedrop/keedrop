context("Mobile layout", function () {
  // Space left for the page while the on-screen keyboard is open
  const devices = [
    { name: "small phone in portrait", width: 320, height: 210 },
    { name: "phone in landscape", width: 667, height: 170 }
  ];

  devices.forEach(function(device) {
    it("Keeps the header visible while the secret input is focused on a " + device.name, function() {
      cy.viewport(device.width, device.height);
      cy.visit("/");
      cy.get("#secret").focus();
      cy.get("#secret").should("have.focus");
      cy.window().then(function(win) {
        const heading = win.document.querySelector("header .tagline h2").getBoundingClientRect();
        const input = win.document.getElementById("secret").getBoundingClientRect();
        expect(heading.top, "header top").to.be.at.least(0);
        expect(input.bottom, "input bottom").to.be.at.most(win.innerHeight);
      });
    });
  });
});
