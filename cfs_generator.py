import docker
import os
import subprocess
import sys

# argv1: -c = custom_img, -p = premade_image
# argv2: desired_image_name (e.g. cfs-ubuntu)
# argv3: 
#	if -p: premade image name (e.g. ubuntu)
#	if -c: path to dockerfile

# e.g. sudo python3 cfs_generator.py -p ubuntu-cfs ubuntu
# e.g. sudo python3 cfs_generator.py -c test-cfs /usr/bin/

ZAR_TOOL = "./main-bf"
dockerfile = "FROM scratch"
client = docker.from_env()

if __name__ == "__main__":
	output_img_name = sys.argv[2]
	img_name = None
	img_path = None

	# Do we need to build the image?
	if (sys.argv[1] == "-c"):
		print("Building image..")
		img_path = sys.argv[3]
		img_name = "init-" + str(output_img_name)

		client.images.build(path = img_path, tag=img_name)
	
	else:
		img_name = sys.argv[3]

		# Do we need to download the image?
		if (len(client.images.list(img_name)) == 0):
			print("Downloading image..")
			client.images.pull(img_name)


	tmp_dir = "/tmp/" + img_name + "/"
	
	try:
		os.mkdir(tmp_dir)
	except(FileExistsError):
		pass

	# Connect to the image
	cli = docker.APIClient(base_url='unix://var/run/docker.sock')
	image_info = cli.inspect_image(img_name)

	data = image_info["GraphDriver"]["Data"]
	print(image_info)

	print()


	if "LowerDir" in data:
		dirs = data["LowerDir"].split(":")
		dirs.reverse()
	else:
		dirs = []
		print("no lower dir in image", img_name)

	dirs.append(data["UpperDir"])
	print("\n".join(dirs))
	
	layer_count = 0
	img_history = client.images.get(img_name).history()
	img_history.reverse()

	for layer_dir in dirs:
		print("Looping through layer: ", layer_dir)

		tokens = layer_dir.split("/")
		id = tokens[-2]
		output_file_base_name = "{}.img".format(id)
		output_file_base_name_w_order = "{}.img".format(layer_count)
		output_file = os.path.join(tmp_dir, output_file_base_name)
		subprocess.run([ZAR_TOOL, "-w", "-dir=" + layer_dir, "-o=" + output_file, "-pagealign"])
		dockerfile += "\nADD {} /{}".format(output_file_base_name, output_file_base_name_w_order)
		layer_count += 1


	# Get command on file
	if (sys.argv[1] == "-c"):
		with open(os.path.join(img_path, "Dockerfile"), "r") as f:
			lines = f.readlines()

			if ("CMD" in lines[-1]):
				# These are not created in layers
				print("Adding command to Dockerfile: ", lines[-1])
				dockerfile += "\n" + lines[-1]
	
	print(dockerfile)

	# Generate new dockerfile
	with open(os.path.join(tmp_dir, "Dockerfile"), "w") as fw:
		fw.write(dockerfile)


	# Create new image
	print("Building cfs image..")
	client.images.build(path = tmp_dir, tag=output_img_name)

	print("Created cfs image with tag:", output_img_name)
