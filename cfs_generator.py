import docker
import os
import subprocess
import sys
# argv1: image name, argv2: new image location
# e.g. sudo python3 main.py test_image /tmp/cfs_img

ZAR_TOOL = "./main"
dockerfile = "FROM scratch"

if __name__ == "__main__":
    cli = docker.APIClient(base_url='unix://var/run/docker.sock')
    image_info = cli.inspect_image(sys.argv[1])
    # print(image_info)
    data = image_info["GraphDriver"]["Data"]
    if "LowerDir" in data:
        dirs = data["LowerDir"].split(":")
        dirs.reverse()
    else:
        dirs = []
        print("no lower dir in image", sys.argv[1])
    dirs.append(data["UpperDir"])
    print("\n".join(dirs))
    layer_count = 0
    for layer_dir in dirs:
        tokens = layer_dir.split("/")
        id = tokens[-2]
        output_file_base_name = "{}.img".format(id)
        output_file_base_name_w_order = "{}.img".format(layer_count)
        output_file = os.path.join(sys.argv[2], output_file_base_name)
        subprocess.run([ZAR_TOOL, "-w", "-dir=" + layer_dir, "-o=" + output_file, "-pagealign"])
        dockerfile += "\nADD {} /{}".format(output_file_base_name, output_file_base_name_w_order)
        layer_count += 1
    print(dockerfile)
    with open(os.path.join(sys.argv[2], "Dockerfile"), "w") as fw:
        fw.write(dockerfile)


