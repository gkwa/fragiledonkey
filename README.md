# fragiledonkey

Purpose:

One-off to delete test images matching pattern.



## Usage example

```bash
# query for images in us-west-2 matching name tag pattern `northflier-???-??-??`
fragiledonkey query

# delete the ones older than 7d
fragiledonkey cleanup --older-than 7d
```
