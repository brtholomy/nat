import csv

from selenium import webdriver
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.common.by import By


def WriteFile(name, content):
    f = open("./out/" + name, "w")
    f.write(content)
    f.close()

baseurl = "http://www.nietzschesource.org/{}/print"

driver = webdriver.Firefox()

with open('./in/menu.csv', newline='') as csvfile:
    # don't use , because the URLs have commas:
    menu = csv.reader(csvfile, delimiter=' ')
    for row in menu:
        target = row[0]
        url = baseurl.format(target)
        name = target[6:] + ".html"
        print(f'{url = }')

        driver.get(url)
        element = driver.find_element(By.CSS_SELECTOR, "div#visore")
        WriteFile(name, element.get_attribute('innerHTML'))

driver.quit()
