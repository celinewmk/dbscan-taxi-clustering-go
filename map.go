// Project CSI2120/CSI2520
// Winter 2022
// Prof: Robert Laganiere, uottawa.ca
// Projet Intégrateur: Partie Go
// Céline Wan Min Kee
// Numéro étudiant: 300193369

package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// Structure GPScoord pour les points du clustering
type GPScoord struct {
	lat  float64
	long float64
}

// Structure LabelledGPScoord pour les points du clustering avec leur label
type LabelledGPScoord struct {
	GPScoord
	ID    int // point ID
	Label int // cluster ID
}

// Structure Job utilisé par le producteur pour envoyer l'algo dbscan à un channel
type Job struct {
	coords []LabelledGPScoord
	minPts int
	eps    float64
	ID     int
}

// Constante
const N int = 4               // division de la grille NxN (peut être changé)
const ConsumerThreads int = 4 // nombre de fils consommateurs (peut être changé)
const MinPts int = 5          // nombre minimum de voisins qu'un point peut avoir
const eps float64 = 0.0003    // valeur epsilon à utiliser
const filename string = "yellow_tripdata_2009-01-15_9h_21h_clean.csv"

// Fonction main
func main() {

	start := time.Now()

	gps, minPt, maxPt := readCSVFile(filename)
	fmt.Printf("Number of points: %d\n", len(gps))

	minPt = GPScoord{40.7, -74.}
	maxPt = GPScoord{40.8, -73.93}

	// geographical limits
	fmt.Printf("SW:(%f , %f)\n", minPt.lat, minPt.long)
	fmt.Printf("NE:(%f , %f) \n\n", maxPt.lat, maxPt.long)

	// Parallel DBSCAN STEP 1.
	incx := (maxPt.long - minPt.long) / float64(N)
	incy := (maxPt.lat - minPt.lat) / float64(N)

	var grid [N][N][]LabelledGPScoord // a grid of GPScoord slices

	// Create the partition
	// triple loop! not very efficient, but easier to understand

	partitionSize := 0
	for j := 0; j < N; j++ {
		for i := 0; i < N; i++ {

			for _, pt := range gps {

				// is it inside the expanded grid cell
				if (pt.long >= minPt.long+float64(i)*incx-eps) && (pt.long < minPt.long+float64(i+1)*incx+eps) && (pt.lat >= minPt.lat+float64(j)*incy-eps) && (pt.lat < minPt.lat+float64(j+1)*incy+eps) {

					grid[i][j] = append(grid[i][j], pt) // add the point to this slide
					partitionSize++
				}
			}
		}
	}

	// ***
	// This is the non-concurrent procedural version
	// It should be replaced by a producer thread that produces jobs (partition to be clustered)
	// And by consumer threads that clusters partitions
	// for j := 0; j < N; j++ {
	// 	for i := 0; i < N; i++ {

	// 		DBscan(grid[i][j], MinPts, eps, i*10000000+j*1000000)
	// 	}
	// }

	// Parallel DBSCAN STEP 2.
	// Apply DBSCAN on each partition
	// ...

	fmt.Printf("Partition with N = %v and %v consumer threads\n", N, ConsumerThreads)
	fmt.Println()

	// création d'un channel pour envoyer des Job
	jobs := make(chan Job, 5)

	// mutex pour la synchronisation
	var mutex sync.WaitGroup
	mutex.Add(ConsumerThreads)

	// Utilisation du patron de design producteur/consommateur

	// appelle les fils consommateurs
	for i := 0; i < ConsumerThreads; i++ {
		go consomme(jobs, &mutex)
	}

	// producteur qui envoie les tâche à réaliser: l'algorithme DBSCAN sur chaque partition
	for j := 0; j < N; j++ {
		for i := 0; i < N; i++ {
			jobs <- Job{grid[i][j], MinPts, eps, i*10000000 + j*1000000}
		}
	}

	// fermeture du channel
	close(jobs)

	// on attend les fils consommateurs
	mutex.Wait()

	// Parallel DBSCAN step 3.
	// merge clusters
	// *DO NOT PROGRAM THIS STEP

	end := time.Now()
	fmt.Printf("\nExecution time: %s of %d points\n", end.Sub(start), partitionSize)
	fmt.Printf("Number of CPUs: %d", runtime.NumCPU())
}

// Fonction consommateur qui applique l'agorithme DBSCAN sur chaque partition
// jobs: channel utilisé qui traite les tâches à réaliser
// done: pointeur au mutex
func consomme(jobs chan Job, done *sync.WaitGroup) {
	for {
		j, more := <-jobs
		if more { // si il y a toujours des Job dans le channel
			DBscan(j.coords, j.minPts, j.eps, j.ID) // appelle de DBSCAN sur la tâche (Job) j
		} else { // s'il n'y a plus de Job
			done.Done() // signale la fin de le fil consommateur
			return
		}
	}
}

// Applies DBSCAN algorithm on LabelledGPScoord points
// LabelledGPScoord: the slice of LabelledGPScoord points
// MinPts, eps: parameters for the DBSCAN algorithm
// offset: label of first cluster (also used to identify the cluster)
// returns number of clusters found
func DBscan(coords []LabelledGPScoord, MinPts int, eps float64, offset int) (nclusters int) {

	nclusters = 0 // initialise le cluster counter
	for i := 0; i < len(coords); i++ {

		if coords[i].Label != 0 { // point déjà traité (0 représente "undefined")
			continue
		}

		neighbours := findNeighbours(coords, &coords[i], eps) // trouve les voisins

		if len(neighbours) < MinPts { // vérifie la densité
			coords[i].Label = -1 //défini le label à -1 qu'on utilise comme "Noise"
			continue
		}

		nclusters++                          // label du prochain cluster
		coords[i].Label = nclusters + offset // défini le label du point initial en rajoutant le offset

		var SeedSet []*LabelledGPScoord          // création d'un slice de pointers pour le cluster
		SeedSet = append(SeedSet, neighbours...) //Ajout les voisins dans le cluster dans le point initial

		for j := 0; j < len(SeedSet); j++ { // traverse le cluster pour l'étendre

			if SeedSet[j].Label == -1 {
				SeedSet[j].Label = nclusters + offset // changement de -1 (Noise) à un point de bord
			}
			if SeedSet[j].Label != 0 { // point déjà traité
				continue
			}

			SeedSet[j].Label = nclusters + offset // label voisin

			N := findNeighbours(coords, SeedSet[j], eps) // trouve les voisins

			if len(N) >= MinPts { // vérifie la densité si c'est un point central
				SeedSet = append(SeedSet, N...) // ajoute les nouveaux voisins au cluster
			}
		}
	}

	// End of DBscan function
	// Printing the result (do not remove)
	fmt.Printf("Partition %10d : [%4d,%6d]\n", offset, nclusters, len(coords))

	return nclusters
}

// Fonction qui trouve les voisins d'un point
// coords: une slice de LabelledGPScoord contenant tous les points à traiter
// Q: un pointeur vers le point dont on doit trouver ses voisins
// eps: la valeur epsilon à utiliser
// retourne un slice de pointeur à chaque voisin
func findNeighbours(coords []LabelledGPScoord, Q *LabelledGPScoord, eps float64) []*LabelledGPScoord {
	var neighbours []*LabelledGPScoord // création d'un slice pour les voisins
	for i := 0; i < len(coords); i++ { // traverse toute la base de donnée
		// calcule la distance et vérifie epsilon en utilisant la fonction distFunt
		// verifie également qu'on ne rajoute pas le point Q lui-même dans les voisins (coords[i].ID != *&Q.ID)
		if distFunc(Q, &coords[i]) <= eps && coords[i].ID != *&Q.ID {
			neighbours = append(neighbours, &coords[i]) // ajout à la liste de voisins
		}
	}
	return neighbours
}

// Fonction qui calcule la distance euclidienne entre 2 points
// Q: un pointeur vers le point Q
// P: un pointeur vers le point P
// retourne la distance euclidienne entre P et Q
func distFunc(Q *LabelledGPScoord, P *LabelledGPScoord) float64 {
	// utilise la formule sqrt((x - x')^2 + (y - y')^2) pour calculer la distance euclidienne
	return math.Sqrt((Q.lat-P.lat)*(Q.lat-P.lat) + (Q.long-P.long)*(Q.long-P.long))
}

// reads a csv file of trip records and returns a slice of the LabelledGPScoord of the pickup locations
// and the minimum and maximum GPS coordinates
func readCSVFile(filename string) (coords []LabelledGPScoord, minPt GPScoord, maxPt GPScoord) {

	coords = make([]LabelledGPScoord, 0, 5000)

	// open csv file
	src, err := os.Open(filename)
	defer src.Close()
	if err != nil {
		panic("File not found...")
	}

	// read and skip first line
	r := csv.NewReader(src)
	record, err := r.Read()
	if err != nil {
		panic("Empty file...")
	}

	minPt.long = 1000000.
	minPt.lat = 1000000.
	maxPt.long = -1000000.
	maxPt.lat = -1000000.

	var n int = 0

	for {
		// read line
		record, err = r.Read()

		// end of file?
		if err == io.EOF {
			break
		}

		if err != nil {
			panic("Invalid file format...")
		}

		// get lattitude
		lat, err := strconv.ParseFloat(record[9], 64)
		if err != nil {
			panic("Data format error (lat)...")
		}

		// is corner point?
		if lat > maxPt.lat {
			maxPt.lat = lat
		}
		if lat < minPt.lat {
			minPt.lat = lat
		}

		// get longitude
		long, err := strconv.ParseFloat(record[8], 64)
		if err != nil {
			panic("Data format error (long)...")
		}

		// is corner point?
		if long > maxPt.long {
			maxPt.long = long
		}

		if long < minPt.long {
			minPt.long = long
		}

		// add point to the slice
		n++
		pt := GPScoord{lat, long}
		coords = append(coords, LabelledGPScoord{pt, n, 0})
	}

	return coords, minPt, maxPt
}
