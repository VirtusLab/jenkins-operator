package theme

// SetThemeGroovyScript it's a groovy script which set custom jenkins theme
// TODO move to base configuration
var SetThemeGroovyScript = `
import jenkins.*
import jenkins.model.*
import hudson.*
import hudson.model.*
import org.jenkinsci.plugins.simpletheme.ThemeElement
import org.jenkinsci.plugins.simpletheme.CssTextThemeElement
import org.jenkinsci.plugins.simpletheme.CssUrlThemeElement

Jenkins jenkins = Jenkins.getInstance()

def decorator = Jenkins.instance.getDescriptorByType(org.codefirst.SimpleThemeDecorator.class)

List<ThemeElement> configElements = new ArrayList<>();
configElements.add(new CssTextThemeElement("DEFAULT"));
configElements.add(new CssUrlThemeElement("https://cdn.rawgit.com/afonsof/jenkins-material-theme/gh-pages/dist/material-light-green.css"));
decorator.setElements(configElements);
decorator.save();

jenkins.save()
`
